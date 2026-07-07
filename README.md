# cis-examples

This project is a collection of examples for the u-space data provisionng and exchange between the CIS and USS.

## Authentication

This project uses OAuth 2.0 with the Client Credentials Grant flow for authentication.
To interact with the examples, you must provide a valid Client ID and Client Secret issued by the OpenID Connect (OIDC) provider.

Required parameters:

```
  -oidc-client-id string
        oidc client id
  -oidc-client-secret string
        oidc client secret
  -oidc-token-url string
        url of the authentication server, token endpoint expected, protocol expected
```

## Geo-Awareness

### Description

This example demonstrates how to discover and ingest active ED-318 geozones within a specific geographic area.
The client queries the DSS for active constraints referneces, fetches the corresponding constraints details from 
their managers and prints the embeded ED-318 geozone.

### Usage

```sh
go build -o geoawareness ./cmd/geoawareness
```

then run `./geoawareness` with
```
  -dss-url string
        base url of the dss, expect protocol to be part of it
  -view string
        lat1,lng1,lat2,lng2 each as float
```

## Surveillance

### Description

This example demonstrates how to discover and stream real-time ATM data within a specific geographic area.
The client queries the DSS for active Traffic Surveilled Areas (TSA), and consumes the Server-Sent Event (SSE) flight streams
from individual surveillance providers.

![til](./docs/surveillance-example.gif)

### Usage

```sh
go build -o surveillance ./cmd/surveillance
```

then run `./surveillance` with
```
  -dss-url string
        base url of the dss, expect protocol to be part of it
  -view string
        lat1,lng1,lat2,lng2 each as float
```
