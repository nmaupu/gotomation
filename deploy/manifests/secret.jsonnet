local k = import 'github.com/jsonnet-libs/k8s-libsonnet/1.30/main.libsonnet';
local v = import 'values.libsonnet';

local s = k.core.v1.secret;

s.new('gotomation', null)
+ s.withData({
  token: std.base64(v.gotomation.hassToken),
  config: std.base64(v.gotomation.config.repo),
})
+ s.metadata.withLabels({
  test: 'test',
})
