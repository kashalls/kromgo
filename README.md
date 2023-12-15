# Kromgo

A simple go project that allows you to expose prometheus metrics "safely" to a public source. Uses the official prometheus go api client. Better than exposing a grafana image rendering instance to the WWW.

It allows you to define your own metric names and your own prometheus queries as long as they return a single value at the end. There is config support to allow you to format the response with strings before and after the value.

You can use [shields.io](https://sheilds.io) and use either the [Dynamic JSON Badge](https://shields.io/badges/dynamic-json-badge) or the [Endpoint Badge](https://shields.io/badges/endpoint-badge) and add dynamic coloring with ranges you set.

[Config Example](./config.yaml.example)

## Performance

Queries take around 5ms ~ 75ms to complete depending on how many breaks my prometheus server takes. This was running on my [home-cluster](https://github.com/kashalls/home-cluster) and runs 3 instances, so depending on the query YMMV.

## Example Request

### Endpoint Response

This format is provided to support Shield.io's [Endpoint Badge](https://shields.io/badges/endpoint-badge) endpoint.

`HTTP GET localhost:8080/query?format=endpoint&metric=node_cpu_usage`

```json
{
    "color": "green",
    "label": "node_cpu_usage",
    "message": "17.5",
    "schemaVersion": 1
}
```

### Raw Response

`HTTP GET localhost:8080/query?metric=node_cpu_usage`

```json
[
    {
        "metric": {},
        "value": [
            1702664619.78,
            "17.5"
        ]
    }
]
```

### 🤝 Gratitude and Thanks

Thanks to all of the people at the [Home Operations](https://discord.gg/home-operations) Discord community. Be sure to check it out, its a blast!
