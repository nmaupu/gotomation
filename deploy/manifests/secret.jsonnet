local k = import 'github.com/jsonnet-libs/k8s-libsonnet/1.30/main.libsonnet';
local v = std.parseYaml(importstr 'values.yaml');

local s = k.core.v1.secret;

s.new('gotomation', null)
+ s.withData({
  token: std.base64(v.hassToken),
  config: std.base64(v.configRepo),
})
+ s.metadata.withLabels({
  test: 'test',
})
