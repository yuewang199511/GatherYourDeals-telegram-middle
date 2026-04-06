# Authentication policy

This middleware will call the other GYD services and they may require authentication.

In order to do so, it needs to include this header in all of the requests going into the downstream.

```json
{"Authorization": "Bearer {{access_token}}"}
```

# store authentication

User will receive JWT key and refresh token from service when login.

They will be cached in REDIS with keys as the chat_id.

# initialization

If no corresponding JWT or refresh token can be found, this middleware will call gatherYourDeals-data service to login.

# log out
If user request to logout, middleware will logout and also revoke the cahce in redis

# keep JWT fresh
before sending request to downstream, this client should check JWT key expiration time first. If already expired or will be expired 5 minutes later, refresh it first.