great! it works. Now use sockets and libraries like socket.io to do matchmaking. remember to follow the
  requirements. a rider is shown to the driver if and only if the driver has an active planned trip with one of it
  as a pickup point & has not already passed that pickup point.
  Once all such riders are shown and assigned, please create a room-like booked situation and display- 1. driver's
  location and movement to rider. 2. rider's location and number to driver.

  once the driver reaches the rider, a new 'trip in-progress' relationship exists between them. A seat is
  occupied(new number of seats decreases). show both the path to the destination(nearest metrostation) and the
  movement towards it.

  once the driver reaches the metro station and drops the rider- the room is destroyed- with the past transaction
  details stored for the rider and the driver. A seat is freed, and the past pickup point is erased in further
  matchmaking.

  Please implement this first, so later we can think about scheduling the trips.

  Plan out your actions, if DB schema modifications are required, plan them out and write modified idempotent
  scripts.

  Make required backend and frontend changes. This is a big task, so divide it into phases- DB, backend,
  frontend(ios), frontend(web).
