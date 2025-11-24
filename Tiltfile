# -*- mode: Python -*-

# For more on Extensions, see: https://docs.tilt.dev/extensions.html
load('ext://restart_process', 'docker_build_with_restart')
load('ext://local_output', 'local_output')

default_registry('ghcr.io/vasu1124')
allow_k8s_contexts(['colima', 'Default'])

compile_cmd = 'make introspect-linux'

# print(compile_cmd)

local_resource(
  'introspect-compile',
  compile_cmd,
  deps=['./cmd', './pkg', './vendor'],
  labels=['introspect']
)

docker_build_with_restart(
  'ghcr.io/vasu1124/introspect',
  '.',
  entrypoint=['/introspect-linux'],
  dockerfile='docker/Dockerfile.alpine',
  only=[
    './introspect-linux',
    './css', 
    './tmpl',
  ],
  live_update=[
    sync('./css', '/css'),
    sync('./tmpl', '/tmpl'),
  ],
)

# Load Kubernetes YAML and inject the git tag
yaml = kustomize('./kubernetes/all-in-one')
yaml = blob(str(yaml).replace('image: ghcr.io/vasu1124/introspect:1.1.0', 'image: introspect:' + TAG))
k8s_yaml(yaml)

k8s_resource(
  'introspect', 
  port_forwards=[9090],  
  resource_deps=['introspect-compile'], 
  labels=['introspect']
)

k8s_resource(workload='introspect', objects=[
  'introspect-config:configmap',
  'introspect-tls:secret',
  'introspect-validationwebook:validatingwebhookconfiguration',
  'uselessmachines.introspect.actvirtual.com:customresourcedefinition',
  'introspect-election-role:Role',
  'introspect-election-rolebinding:RoleBinding',
  'uselessmachine-editor-role:clusterrole',
  'uselessmachine-viewer-role:clusterrole',
  'introspect-rolebinding:clusterrolebinding',
  'introspect-secret:secret',],
  labels=['introspect']
)
k8s_resource(workload='mongodb', objects=[
  'mongodb:persistentvolumeclaim',
  'mongodb-secret:secret'],
  labels=['introspect']
)
k8s_resource(workload='etcd', objects=[
  'etcd:secret'],
  labels=['introspect']
)
k8s_resource(workload='valkey', objects=[
  'valkey:persistentvolumeclaim',
  'valkey-secret:secret'],
  labels=['introspect']
)

#v1alpha1.extension_repo(name='tilt-extensions', url='https://github.com/tilt-dev/tilt-extensions')
#v1alpha1.extension(
#  name='ngrok', 
#  repo_name='tilt-extensions', 
#  repo_path='ngrok',
#)
