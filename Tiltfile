# -*- mode: Python -*-

# For more on Extensions, see: https://docs.tilt.dev/extensions.html
load('ext://restart_process', 'docker_build_with_restart')
load('ext://local_output', 'local_output')

VERSION = local_output('cat introspect.VERSION')
COMMIT  = local_output('git rev-parse HEAD')
BRANCH  = local_output('git rev-parse --abbrev-ref HEAD')

LDFLAGS = '-ldflags "-X github.com/vasu1124/introspect/pkg/version.Version=' + VERSION + ' -X github.com/vasu1124/introspect/pkg/version.Commit=' + COMMIT + ' -X github.com/vasu1124/introspect/pkg/version.Branch=' + BRANCH + '"'
compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ' + LDFLAGS + ' -o introspect-linux-amd64 ./cmd/introspect'
print(compile_cmd)

local_resource(
  'introspect-compile',
  compile_cmd,
  deps=['./cmd', './pkg']
)

docker_build_with_restart(
  'vasu1124/introspect',
  '.',
  entrypoint=['/introspect-linux-amd64'],
  dockerfile='docker/Dockerfile.alpine',
  only=[
    './introspect-linux-amd64',
    './css', 
    './tmpl'
  ],
  live_update=[
    sync('./css', '/css'),
    sync('./tmpl', '/tmpl'),
  ],
)

k8s_yaml(kustomize('./kubernetes/all-in-one'))
k8s_resource('introspect', port_forwards=9090, resource_deps=['introspect-compile'])
k8s_resource(new_name='kustomize', objects=[
  'uselessmachines.introspect.actvirtual.com:customresourcedefinition',
  'introspect-lease:role',
  'introspect-role:clusterrole',
  'introspect-lease-rolebinding:rolebinding',
  'introspect-rolebinding:clusterrolebinding',
  'mongodb:persistentvolumeclaim',
  'introspect-config:configmap',
  'mongodb-secret:secret'
])