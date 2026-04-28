## uses case 1: 

### description

in this example, we call the traffic_surveilled_area from the dss, gather the uss url
then query each of them to retrieve the stream of data, and print it during 15s.

### usage:

go build -o example1 ./cmd/example-1-simple-consumer

then run it with the rights flags
```
 -oidc-client-id string
        client id
  -oidc-client-secret string
        client secret
  -dss-audiences string
        dss audience to pass to oidc
  -dss-base-path string
        base path for the dss (default "/surveillance/v0")
  -dss-scopes string
        dss scopes to pass to oidc
  -dss-url string
        base url of the dss
  -oidc-token-url string
        url of the authentication server, token endpoint exected
  -uss-scopes string
        uss scopes to pass to oidc
  -view string
        lat1,lng1,lat2,lng2 each as float
```

./cis-example