Code Generation function used:

Dss part: 

cd $ROOT/api/surveillance/v0/dss
go tool oapi-codegen -o surveillance.types.gen.go -package surveillance_dss_v0 -generate types  -include-tags dss ../surveillance.yaml
go tool oapi-codegen -o surveillance.client.dss.gen.go -package surveillance_dss_v0 -generate client -response-type-suffix HttpResponse -include-tags dss ../surveillance.yaml

Uss part:

cd $ROOT/api/surveillance/v0/uss
go tool oapi-codegen -o surveillance.client.uss.gen.go -package surveillance_uss_v0 -generate client -response-type-suffix HttpResponse -include-tags sp ../surveillance.yaml
go tool oapi-codegen -o surveillance.types.gen.go -package surveillance_uss_v0 -generate types,skip-prune  -include-tags sp ../surveillance.yaml