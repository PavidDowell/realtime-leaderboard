# realtime-leaderboard
This is an example project to make a service realtime by using a comination of, postgres, go, redis 
and a reverse proxy with nginx.


You can create a player with the curl command and assign them a score

curl -X POST http://localhost:8081/score   -H "Content-Type: application/json"   -d '{"username":"david","delta":25,"source":"match_end","idempotencyKey":"req-2"}'

This can be viewed in real time on the test.html page at the root of the project. This will be improved with a dashboard style view that will connect to the websocket to display the information in realtime
