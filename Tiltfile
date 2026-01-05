# -*- mode: Python -*-
# SPDX-FileCopyrightText: 2025 vasu1124
#
# SPDX-License-Identifier: CC0-1.0

# For more on Extensions, see: https://docs.tilt.dev/extensions.html
load('ext://restart_process', 'docker_build_with_restart')
load('ext://local_output', 'local_output')
load('ext://restart_process', 'custom_build_with_restart')

def podman_build(
  ref, context, ignore=None, extra_flags=None, deps=None, live_update=[], push_extra_flags=None
):
  """Use Podman (https://podman.io/) to build images for Tilt.
  Args:
    ref: The name of the image to build. Must match the image
      name in the Kubernetes resources you're deploying.
    context: The build context of the binary to build. Expressed as a file path.
    deps: Changes to the given files or directories that will trigger rebuilds.
      Defaults to the build context.
    ignore: Changes to the given files or directories do not trigger rebuilds.
      Does not affect the build context.
    extra_flags: Extra flags to pass to podman build. Expressed as an argv-style array.
    push_extra_flags: Extra flags to pass to podman push. Expressed as an argv-style array.
    live_update: Set of steps for updating a running container
      (see https://docs.tilt.dev/live_update_reference.html)
  """
  deps = deps or [context]
  extra_flags = extra_flags or []
  push_extra_flags = push_extra_flags or []
  extra_flags_str = ' '.join([shlex.quote(f) for f in extra_flags])
  push_extra_flags_str = ' '.join([shlex.quote(f) for f in push_extra_flags])

  # We use --format=docker due to
  # https://github.com/containers/buildah/issues/1589
  # which lots of people are still reporting, even though it's closed :shrug:
  push_cmd = "podman push %s --format=docker $EXPECTED_REF\n" % push_extra_flags_str

  custom_build(
    ref=ref,
    command=(
      "set -ex\n" +
      "podman build -t $EXPECTED_REF %s %s\n" +
      push_cmd
    ) % (extra_flags_str, shlex.quote(context)),
    ignore=ignore,
    deps=deps,
    live_update=live_update,
    skips_local_docker=True,
  )

def podman_build_with_restart(
    ref, context, entrypoint, ignore=None, extra_flags=None, deps=None, live_update=[], push_extra_flags=None
):
  """Use Podman (https://podman.io/) to build images for Tilt. Wrap a custom_build_with_restart so that the last step
    of any live update is to rerun the given entrypoint.
  Args:
    ref: The name of the image to build. Must match the image
      name in the Kubernetes resources you're deploying.
    context: The build context of the binary to build. Expressed as a file path.
    entrypoint: The command to be (re-)executed when the container starts or when a live_update is run.
    deps: Changes to the given files or directories that will trigger rebuilds.
      Defaults to the build context.
    ignore: Changes to the given files or directories do not trigger rebuilds.
      Does not affect the build context.
    extra_flags: Extra flags to pass to podman build. Expressed as an argv-style array.
    push_extra_flags: Extra flags to pass to podman push. Expressed as an argv-style array.
    live_update: Set of steps for updating a running container
      (see https://docs.tilt.dev/live_update_reference.html)
  """
  deps = deps or [context]
  extra_flags = extra_flags or []
  push_extra_flags = push_extra_flags or []
  extra_flags_str = ' '.join([shlex.quote(f) for f in extra_flags])
  push_extra_flags_str = ' '.join([shlex.quote(f) for f in push_extra_flags])
  # We use --format=docker due to
  # https://github.com/containers/buildah/issues/1589
  # which lots of people are still reporting, even though it's closed :shrug:
  push_cmd = "podman push %s --format=docker $EXPECTED_REF\n" % push_extra_flags_str

  custom_build_with_restart(
    ref=ref,
    command=(
      "set -ex\n" +
      "podman build -t $EXPECTED_REF %s %s\n"
    ) % (extra_flags_str, shlex.quote(context)),
    ignore=ignore,
    deps=deps,
    entrypoint=entrypoint,
    live_update=live_update,
  )

default_registry('ghcr.io/vasu1124')
allow_k8s_contexts(['colima', 'Default', 'desktop', 'docker-desktop', 'kind-kind', 'rancher-desktop'])

# Set the version tag to use for Tilt builds
TAG = str(local('git describe --tags --always --dirty --abbrev=0')).strip()

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
#yaml = blob(str(yaml).replace('image: ghcr.io/vasu1124/introspect:1.1.0', 'image: introspect:' + TAG))
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

