#!/usr/bin/env bash
# Deploy one commit to LastMile's Render services and wait until each is live.
#
#   RENDER_API_KEY=rnd_... scripts/render-deploy.sh <commit-sha> [service ...]
#   scripts/render-deploy.sh --dry-run <commit-sha> [service ...]
#
# Services default to $RENDER_SERVICES, else the full blueprint below. Names
# that do not exist on Render are skipped with a warning, so the same call
# works for the full blueprint and for the single-service `lastmile-stack`.
#
# Order: every backend is triggered at once and awaited together; then the
# gateway tier (lastmile-gateway, lastmile-stack); then the web tier
# (lastmile-web or any static site). A failed tier stops the rollout, so the
# gateway never goes live in front of a backend that failed to deploy.
#
# Git-backed services deploy {"commitId": <sha>}. Image-backed services
# deploy their configured image with the tag replaced by <sha> (CI pushes
# ghcr.io/<owner>/lastmile-<svc>:<sha> on main).
#
# --dry-run never triggers anything: it prints the plan, and if
# RENDER_API_KEY is set it also looks the services up (read-only).
#
# Environment:
#   RENDER_API_KEY         Render API key (required unless --dry-run)
#   RENDER_SERVICES        space-separated default service list
#   RENDER_API_BASE        API root (default https://api.render.com/v1); tests point it at a fake
#   RENDER_DEPLOY_TIMEOUT  seconds to wait for each tier to go live (default 1500 = 25 min)
#   RENDER_POLL_INTERVAL   seconds between status polls (default 15)
#   RENDER_CLEAR_CACHE     "clear" to wipe the build cache first (default do_not_clear)
#
# Needs bash, curl and jq. Exit codes: 0 all live, 1 deploy failed, 2 usage/config error.
set -euo pipefail

DEFAULT_SERVICES="lastmile-user lastmile-driver lastmile-rider lastmile-station lastmile-location
lastmile-matching lastmile-trip lastmile-notification lastmile-gateway lastmile-stack lastmile-web"

API_BASE="${RENDER_API_BASE:-https://api.render.com/v1}"
API_BASE="${API_BASE%/}"
TIMEOUT="${RENDER_DEPLOY_TIMEOUT:-1500}"
INTERVAL="${RENDER_POLL_INTERVAL:-15}"
CLEAR_CACHE="${RENDER_CLEAR_CACHE:-do_not_clear}"

usage() { sed -n '2,31p' "$0" | sed 's/^# \{0,1\}//'; exit 2; }
die() { echo "::error::$*" >&2; exit 2; }
warn() { echo "::warning::$*" >&2; }
fmt_secs() { printf '%dm%02ds' $(($1 / 60)) $(($1 % 60)); }

# ---- arguments ---------------------------------------------------------------
DRY_RUN=0
args=()
for a in "$@"; do
  case $a in
    --dry-run) DRY_RUN=1 ;;
    -h | --help) usage ;;
    -*) die "unknown flag: $a" ;;
    *) args+=("$a") ;;
  esac
done
[ "${#args[@]}" -ge 1 ] || usage
SHA=${args[0]}
[[ $SHA =~ ^[0-9a-f]{7,40}$ ]] || die "not a git commit SHA: '$SHA'"
if [ "${#args[@]}" -gt 1 ]; then
  requested="${args[*]:1}"
else
  requested="${RENDER_SERVICES:-$DEFAULT_SERVICES}"
fi
[[ $INTERVAL =~ ^[0-9]+$ && $TIMEOUT =~ ^[0-9]+$ ]] || die "RENDER_POLL_INTERVAL and RENDER_DEPLOY_TIMEOUT must be whole seconds"
[[ $CLEAR_CACHE == clear || $CLEAR_CACHE == do_not_clear ]] || die "RENDER_CLEAR_CACHE must be 'clear' or 'do_not_clear'"

command -v curl >/dev/null || die "curl is required"
command -v jq >/dev/null || die "jq is required"

# Per-service state lives in parallel indexed arrays (bash 3.2 compatible).
names=() ids=() types=() tiers=() deploys=() states=() notes=() starts=() durations=() urls=() bodies=()
for n in $requested; do
  [[ $n =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid service name: '$n'"
  for existing in ${names[@]+"${names[@]}"}; do [ "$existing" = "$n" ] && continue 2; done
  names+=("$n") ids+=("") types+=("") tiers+=("") deploys+=("") states+=("") notes+=("")
  starts+=(0) durations+=("") urls+=("") bodies+=("")
done
count=${#names[@]}
[ "$count" -gt 0 ] || die "no services to deploy"

tier_of() { # name type -> 1 backend, 2 gateway, 3 web
  case $1 in
    lastmile-gateway | lastmile-stack) echo 2 ;;
    lastmile-web) echo 3 ;;
    *) if [ "$2" = static_site ]; then echo 3; else echo 1; fi ;;
  esac
}
tier_label() { case $1 in 1) echo "backends" ;; 2) echo "gateway" ;; 3) echo "web" ;; esac; }

# ---- Render API ----------------------------------------------------------------
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
if [ -n "${RENDER_API_KEY:-}" ]; then
  # Header file keeps the key out of the process list.
  (umask 077 && printf 'Authorization: Bearer %s\n' "$RENDER_API_KEY" >"$tmp/auth")
elif [ "$DRY_RUN" = 0 ]; then
  die "RENDER_API_KEY is not set (create one under Render > Account Settings > API Keys)"
fi

# api METHOD PATH [JSON] -> sets API_STATUS, body in $API_BODY. Retries 429, 5xx
# and network errors with backoff (honours Retry-After).
API_BODY="$tmp/body"
API_STATUS=000
api() {
  local method=$1 path=$2 data=${3:-} attempt=1 wait
  local curl_args=(-sS -o "$API_BODY" -D "$tmp/headers" -w '%{http_code}' --max-time 30
    -X "$method" -H @"$tmp/auth" -H 'Accept: application/json')
  [ -n "$data" ] && curl_args+=(-H 'Content-Type: application/json' --data "$data")
  while :; do
    : >"$API_BODY"
    API_STATUS=$(curl "${curl_args[@]}" "$API_BASE$path" 2>"$tmp/curl.err") || API_STATUS=000
    case $API_STATUS in
      429 | 5?? | 000)
        if [ "$attempt" -lt 5 ]; then
          wait=$(tr -d '\r' <"$tmp/headers" 2>/dev/null | awk 'tolower($1)=="retry-after:"{print $2}' | tail -1)
          [[ $wait =~ ^[0-9]+$ ]] || wait=$((attempt * attempt * 2))
          [ "$wait" -le 60 ] || wait=60
          echo "  … $method $path -> HTTP $API_STATUS$(head -c 200 "$tmp/curl.err" | tr '\n' ' '); retrying in ${wait}s" >&2
          sleep "$wait"
          attempt=$((attempt + 1))
          continue
        fi
        ;;
    esac
    return 0
  done
}
api_error() { # one-line description of a failed call
  local msg
  msg=$(jq -r '.message // empty' "$API_BODY" 2>/dev/null || true)
  [ -n "$msg" ] || msg=$(head -c 300 "$API_BODY" | tr '\n' ' ')
  [ -n "$msg" ] || msg=$(head -c 300 "$tmp/curl.err")
  echo "HTTP $API_STATUS: $msg"
}

# lookup i -> fills ids/types/urls/bodies or marks the service skipped.
# GET /services?name=<n> returns [{"service": {...}, "cursor": "..."}]; the
# filter is matched again here so only an exact name counts.
lookup() {
  local i=$1 n=${names[$1]} matches
  api GET "/services?name=$n&limit=20"
  [ "$API_STATUS" != 401 ] || die "Render rejected RENDER_API_KEY (HTTP 401); create a new key under Account Settings > API Keys"
  [ "$API_STATUS" = 200 ] || die "looking up $n failed: $(api_error)"
  matches=$(jq --arg n "$n" '[.[] | .service | select(.name == $n)]' "$API_BODY" 2>/dev/null) ||
    die "unexpected response looking up $n: $(head -c 200 "$API_BODY")"
  case $(jq length <<<"$matches") in
    0)
      states[i]=skipped notes[i]="not found on Render"
      missing+=" $n"
      return
      ;;
    1) ;;
    *) die "more than one Render service is named '$n'; rename one so deploys are unambiguous" ;;
  esac
  ids[i]=$(jq -r '.[0].id' <<<"$matches")
  types[i]=$(jq -r '.[0].type' <<<"$matches")
  urls[i]=$(jq -r '.[0].dashboardUrl // ""' <<<"$matches")
  if [ "$(jq -r '.[0].suspended' <<<"$matches")" = suspended ]; then
    states[i]=skipped notes[i]="suspended"
    warn "Render service '$n' is suspended; resume it in the dashboard to deploy it"
    return
  fi
  # Git-backed: pin the commit. Image-backed: same image, tag = commit SHA.
  local image
  image=$(jq -r 'if .[0].repo then "" else (.[0].imagePath // "") end' <<<"$matches")
  if [ -n "$image" ]; then
    image=${image%@*}                              # drop @sha256:digest
    [[ ${image##*/} == *:* ]] && image=${image%:*} # drop :tag (not a registry :port)
    bodies[i]=$(jq -cn --arg u "$image:$SHA" '{imageUrl: $u}')
  else
    bodies[i]=$(jq -cn --arg c "$SHA" --arg cc "$CLEAR_CACHE" '{commitId: $c, clearCache: $cc}')
  fi
  states[i]=planned
}

# trigger i: POST /services/{id}/deploys -> 201 + deploy, or 202 when Render
# queued it behind a running deploy (no body; resolved later by commit).
trigger() {
  local i=$1
  api POST "/services/${ids[i]}/deploys" "${bodies[i]}"
  starts[i]=$(date +%s)
  case $API_STATUS in
    201 | 200)
      deploys[i]=$(jq -r '.id // empty' "$API_BODY" 2>/dev/null || true)
      states[i]=$(jq -r '.status // "created"' "$API_BODY" 2>/dev/null || echo created)
      ;;
    202) states[i]=queued ;;
    *)
      states[i]=failed notes[i]="trigger failed: $(api_error)"
      echo "::error::${names[i]}: ${notes[i]}"
      return
      ;;
  esac
  printf '  %-24s triggered %s (%s)\n' "${names[i]}" "${deploys[i]:-<queued>}" "${states[i]}"
}

# resolve_queued i: find the deploy Render queued for this commit/image.
resolve_queued() {
  local i=$1
  api GET "/services/${ids[i]}/deploys?limit=10"
  [ "$API_STATUS" = 200 ] || return 0
  deploys[i]=$(jq -r --arg s "$SHA" '[.[] | .deploy | select(.commit.id == $s or ((.image.ref // "") | endswith(":" + $s)))][0].id // empty' "$API_BODY" 2>/dev/null || true)
}

# classify STATE -> ok | fail | idle | wait. Render deploy statuses: created,
# queued, build_in_progress, update_in_progress, pre_deploy_in_progress (wait);
# live, deactivated (ok; deactivated = was live, then superseded); build_failed,
# update_failed, pre_deploy_failed, canceled (fail). Unknown statuses keep
# waiting, except anything *_failed. failed/timeout/planned/skipped are ours.
classify() {
  case $1 in
    live | deactivated) echo ok ;;
    canceled | failed | timeout | *_failed) echo fail ;;
    planned | skipped) echo idle ;;
    *) echo wait ;;
  esac
}

# wait_tier T: poll every triggered deploy in tier T until all finish or time out.
wait_tier() {
  local t=$1 deadline i pending status now
  deadline=$(($(date +%s) + TIMEOUT))
  while :; do
    pending=0
    for ((i = 0; i < count; i++)); do
      [ "${tiers[i]}" = "$t" ] && [ "$(classify "${states[i]}")" = wait ] || continue
      [ -n "${deploys[i]}" ] || resolve_queued "$i"
      if [ -n "${deploys[i]}" ]; then
        api GET "/services/${ids[i]}/deploys/${deploys[i]}"
        if [ "$API_STATUS" = 200 ]; then
          status=$(jq -r '.status // "unknown"' "$API_BODY" 2>/dev/null || echo unknown)
          if [ "$status" != "${states[i]}" ]; then
            printf '  %-24s %-22s %s\n' "${names[i]}" "$status" "$(fmt_secs $(($(date +%s) - starts[i])))"
            states[i]=$status
          fi
        else
          echo "  ${names[i]}: status check $(api_error); will retry" >&2
        fi
      fi
      case $(classify "${states[i]}") in
        ok | fail) durations[i]=$(fmt_secs $(($(date +%s) - starts[i]))) ;;
        *) pending=$((pending + 1)) ;;
      esac
    done
    [ "$pending" -gt 0 ] || return 0
    now=$(date +%s)
    if [ "$now" -ge "$deadline" ]; then
      for ((i = 0; i < count; i++)); do
        [ "${tiers[i]}" = "$t" ] && [ "$(classify "${states[i]}")" = wait ] || continue
        notes[i]="still ${states[i]} after $(fmt_secs "$TIMEOUT"); see the Render dashboard"
        states[i]=timeout durations[i]=$(fmt_secs $((now - starts[i])))
        echo "::error::${names[i]}: ${notes[i]}"
      done
      return 0
    fi
    sleep "$INTERVAL"
  done
}

detail() { # time and note of service $1, space-joined
  local d=${durations[$1]} n=${notes[$1]}
  echo "$d${d:+${n:+ }}$n"
}

summary() {
  local i result
  echo
  echo "Render deploy of ${SHA:0:12}:"
  printf '  %-24s %-20s %-28s %s\n' SERVICE RESULT DEPLOY TIME/NOTE
  for ((i = 0; i < count; i++)); do
    result=${states[i]}
    [ "$result" = planned ] && result=$([ "$DRY_RUN" = 1 ] && echo "would deploy" || echo "not started")
    printf '  %-24s %-20s %-28s %s\n' "${names[i]}" "$result" "${deploys[i]:--}" "$(detail "$i")"
  done
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
      echo "### Render deploy of \`${SHA:0:12}\`$([ "$DRY_RUN" = 1 ] && echo " (dry run)")"
      echo
      echo "| Service | Result | Deploy | Time / note |"
      echo "|---|---|---|---|"
      for ((i = 0; i < count; i++)); do
        result=${states[i]}
        [ "$result" = planned ] && result=$([ "$DRY_RUN" = 1 ] && echo "would deploy" || echo "not started")
        local svc=${names[i]}
        [ -n "${urls[i]}" ] && svc="[${names[i]}](${urls[i]})"
        echo "| $svc | $result | ${deploys[i]:--} | $(detail "$i") |"
      done
    } >>"$GITHUB_STEP_SUMMARY"
  fi
}

# ---- plan ------------------------------------------------------------------------
echo "▸ Render deploy of commit $SHA$([ "$DRY_RUN" = 1 ] && echo " (dry run)")"
if [ -f "$tmp/auth" ]; then
  echo "▸ looking up ${count} service(s) at $API_BASE"
  missing=""
  for ((i = 0; i < count; i++)); do lookup "$i"; done
  [ -z "$missing" ] || warn "skipping service(s) not found in this Render workspace:$missing"
else
  echo "  RENDER_API_KEY not set: planning by name only, no API calls"
  for ((i = 0; i < count; i++)); do states[i]=planned; done
fi
found=0
for ((i = 0; i < count; i++)); do
  tiers[i]=$(tier_of "${names[i]}" "${types[i]}")
  [ "${states[i]}" = planned ] && found=$((found + 1))
done
for t in 1 2 3; do
  line=""
  for ((i = 0; i < count; i++)); do
    [ "${tiers[i]}" = "$t" ] && [ "${states[i]}" = planned ] && line+=" ${names[i]}${ids[i]:+ (${ids[i]})}"
  done
  [ -z "$line" ] || echo "  tier $t, $(tier_label "$t"):$line"
done

if [ "$found" -eq 0 ]; then
  summary
  echo "::error::none of the requested services exist in this Render workspace (requested: ${names[*]})" >&2
  exit 2
fi
if [ "$DRY_RUN" = 1 ]; then
  summary
  echo "dry run: nothing was deployed"
  exit 0
fi

# ---- deploy, tier by tier -------------------------------------------------------
failed=0
for t in 1 2 3; do
  members=0
  for ((i = 0; i < count; i++)); do [ "${tiers[i]}" = "$t" ] && [ "${states[i]}" = planned ] && members=$((members + 1)); done
  [ "$members" -gt 0 ] || continue
  echo "▸ tier $t ($(tier_label "$t")): triggering $members deploy(s)"
  for ((i = 0; i < count; i++)); do
    [ "${tiers[i]}" = "$t" ] && [ "${states[i]}" = planned ] && trigger "$i"
  done
  echo "▸ tier $t: waiting for live (timeout $(fmt_secs "$TIMEOUT"), polling every ${INTERVAL}s)"
  wait_tier "$t"
  for ((i = 0; i < count; i++)); do
    [ "${tiers[i]}" = "$t" ] && [ "$(classify "${states[i]}")" = fail ] || continue
    failed=1
    if [ -z "${notes[i]}" ]; then
      notes[i]="see the deploy logs in the Render dashboard"
      echo "::error::${names[i]}: deploy ${deploys[i]} ended ${states[i]}; ${notes[i]}"
    fi
  done
  if [ "$failed" = 1 ]; then
    for ((i = 0; i < count; i++)); do
      [ "${states[i]}" = planned ] && notes[i]="an earlier tier failed"
    done
    break
  fi
done

summary
exit "$failed"
