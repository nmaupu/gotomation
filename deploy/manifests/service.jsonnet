local k = import 'github.com/jsonnet-libs/k8s-libsonnet/1.30/main.libsonnet';
local g = import 'globals.libsonnet';
local v = import 'values.libsonnet';

local svc = k.core.v1.service;

svc.newWithoutSelector('gotomation')
+ svc.metadata.withLabels(g.labels)
+ svc.spec.withPorts({
  port: 80,
  protocol: 'TCP',
  targetPort: g.containerPort,
})
+ svc.spec.withSelector(g.labels)
