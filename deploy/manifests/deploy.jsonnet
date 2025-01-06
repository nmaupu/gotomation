local k = import 'github.com/jsonnet-libs/k8s-libsonnet/1.30/main.libsonnet';

local g = import 'globals.libsonnet';
local v = import 'values.libsonnet';

local d = k.apps.v1.deployment;
local c = k.core.v1.container;

local volumeMountConfig =
  k.core.v1.volumeMount.withMountPath('/config')
  + k.core.v1.volumeMount.withName('config');

local initContainer =
  c.withName('init')
  + c.withImage('%s:%s' % [v.git.image, v.git.tag])
  + c.withImagePullPolicy(v.git.pullPolicy)
  + c.withCommand([
    'bash',
    '-c',
    |||
      set -e
      set -o pipefail
      set -x
      REPO="%s"
      BRANCH="%s"

      date
      git clone "https://$REPO" /tmp/gotomation-config
      rsync -ai --checksum --omit-dir-times --no-p --delete /tmp/gotomation-config/ /config
      cd /tmp/gotomation-config
      git checkout "$BRANCH"
    ||| % [
      v.git.gotomationConfig.repo,
      v.git.gotomationConfig.branch,
    ],
  ])
  + c.withVolumeMounts(volumeMountConfig);

local gitRefresherContainer =
  c.withName('git-refresher')
  + c.withImage('%s:%s' % [v.git.image, v.git.tag])
  + c.withImagePullPolicy(v.git.pullPolicy)
  + c.withCommand([  // This script refresh the git repository configured branch at a regular interval
    'bash',
    '-c',
    |||
      set -e
      set -o pipefail
      set -x

      REPO="%s"
      BRANCH="%s"
      INTERVAL="%d"

      date
      git clone "https://$REPO" /tmp/gotomation-config
      cd /tmp/gotomation-config
      while [ 1 ]; do
        date
        git fetch --all && git reset --hard origin/"$BRANCH"
        rsync -ai --checksum --omit-dir-times --no-p --delete /tmp/gotomation-config/ /config
        sleep "$INTERVAL"
      done
    ||| % [
      v.git.gotomationConfig.repo,
      v.git.gotomationConfig.branch,
      v.git.gotomationConfig.refreshIntervalSeconds,
    ],
  ])
  + c.withVolumeMounts(volumeMountConfig);

local mainContainer =
  c.withName('gotomation')
  + c.withImage('%s:%s' % [v.image.repository, v.image.tag])
  + c.withImagePullPolicy(v.image.pullPolicy)
  + c.withCommand(['gotomation'])
  + c.withArgs([
    '--config=/config/gotomation.yaml',
  ])
  + c.withWorkingDir('/config')
  + c.withEnv([
    {name: 'TZ', value: v.timezone}
  ])
  + c.withVolumeMounts(volumeMountConfig + k.core.v1.volumeMount.withReadOnly(true))
  + (if std.objectHas(v, 'existingSecretEnvVars') && std.length(v.existingSecretEnvVars) > 0 then
       c.withEnvFrom(k.core.v1.envFromSource.secretRef.withName(v.existingSecretEnvVars))
     else
       {})
  + c.livenessProbe.withInitialDelaySeconds(3)
  + c.livenessProbe.withPeriodSeconds(3)
  + c.livenessProbe.httpGet.withPath('/health')
  + c.livenessProbe.httpGet.withPort(6265)
  + c.readinessProbe.withInitialDelaySeconds(1)
  + c.readinessProbe.withPeriodSeconds(2)
  + c.readinessProbe.httpGet.withPath('/health-ex')
  + c.readinessProbe.httpGet.withPort(6265)
  + c.withPorts([{
    containerPort: g.containerPort,
  }]);

d.new(
  'gotomation',
  replicas=1,
  containers=[
    mainContainer,
    gitRefresherContainer,
  ],
)
+ d.metadata.withLabels(g.labels)
+ d.spec.template.spec.withInitContainers(initContainer)
+ d.spec.selector.withMatchLabels({
  'app.kubernetes.io/name': 'gotomation',
})
+ d.spec.template.metadata.withLabels(g.labels)
+ d.spec.template.spec.withVolumes(k.core.v1.volume.fromEmptyDir('config'))
