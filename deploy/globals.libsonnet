local v = std.parseYaml(importstr 'values.yaml');

{
  labels: {
    'app.kubernetes.io/name': 'gotomation',
    'app.kubernetes.io/version': v.image.tag,
  },
}
