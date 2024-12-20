local v = import 'values.libsonnet';

{
  labels: {
    'app.kubernetes.io/name': 'gotomation',
    'app.kubernetes.io/version': v.image.tag,
  },
}
