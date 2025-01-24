local k = import 'github.com/jsonnet-libs/k8s-libsonnet/1.30/main.libsonnet';
local g = import 'globals.libsonnet';
local v = import 'values.libsonnet';

local pvc = k.core.v1.persistentVolumeClaim;

pvc.new('gotomation-config')
+ pvc.metadata.withLabels(g.labels)
+ pvc.spec.withStorageClassName(v.gotomation.config.pvc.storageClassName)
+ pvc.spec.withAccessModes('ReadWriteOnce')
+ pvc.spec.resources.withRequests(
  { storage: v.gotomation.config.pvc.size }
)
