local k = import 'github.com/jsonnet-libs/k8s-libsonnet/1.30/main.libsonnet';
local v = std.parseYaml(importstr 'values.yaml');
local g = import 'globals.libsonnet';

local sts = k.apps.v1.statefulSet;
local c = k.core.v1.container;

local volumeMounts = [
  k.core.v1.volumeMount.withMountPath('/config')
  + k.core.v1.volumeMount.withName('config')
  + k.core.v1.volumeMount.withReadOnly(true),
];

local initContainers = [
  c.withName('init')
  + c.withImage('busybox:latest')
  + c.withArgs([])  // TODO: clone git repo, how to refresh ?
  + c.withVolumeMounts(volumeMounts),
];

local senderConfigFunc(x) = '--senderConfig=%s' % [std.base64(std.manifestJson(x))];
local containers = [
  c.withName('gotomation')
  + c.withImage('%s:%s' % [v.image.repository, v.image.tag])
  + c.withArgs(
    [
      '--config=/config/gotomation.yaml',
      '--token=%s' % [v.gotomation.hassToken],
    ]
    + std.map(senderConfigFunc, v.gotomation.senderConfigs)
  )
  + c.withVolumeMounts(volumeMounts),
];

sts.new(
  'gotomation',
  replicas=1,
  containers=containers,
)
+ sts.metadata.withLabels(g.labels)
+ sts.spec.selector.withMatchLabels(g.labels)
+ sts.spec.template.metadata.withLabels(g.labels)
+ sts.spec.template.spec.withInitContainers(initContainers)
+ sts.spec.template.spec.withVolumes(k.core.v1.volume.fromEmptyDir('config'))
