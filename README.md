uses: 

go build -o cis-examples .

then run it with the rights flags
```
 -client-id string
        client id
  -client-secret string
        client secret
  -dss-audiences string
        dss audience to pass to oidc
  -dss-base-path string
        base path for the dss (default "/surveillance/v0")
  -dss-scopes string
        dss scopes to pass to oidc
  -dss-url string
        base url of the dss
  -oidc-url string
        url of the authentication server, token endpoint exected
  -uss-audiences string
        uss audience to pass to oidc
  -uss-base-path string
        base path of the uss (default "/surveillance/v0")
  -uss-scopes string
        uss scopes to pass to oidc
  -uss-url string
        base url of the uss
  -view string
        lat1,lng1,lat2,lng2 each as float
```

./cis-example