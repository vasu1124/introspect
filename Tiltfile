# -*- mode: Python -*-
# SPDX-FileCopyrightText: 2025 vasu1124
#
# SPDX-License-Identifier: CC0-1.0

# For more on Extensions, see: https://docs.tilt.dev/extensions.html
load('ext://restart_process', 'docker_build_with_restart')
load('ext://local_output', 'local_output')
load('ext://restart_process', 'custom_build_with_restart')

def podman_build(
  ref, context, entrypoint=None, dockerfile=None, ignore=None, deps=None, live_update=[]
):
  """Use Podman (https://podman.io/) to build images for Tilt.
  Args:
    ref: The name of the image to build. Must match the image
      name in the Kubernetes resources you're deploying.
    context: The build context of the binary to build. Expressed as a file path.
    entrypoint: The command to be (re-)executed when the container starts or when a live_update is run.
    dockerfile: The path to the Dockerfile to use for the build.
    deps: Changes to the given files or directories that will trigger rebuilds.
      Defaults to the build context.
    ignore: Changes to the given files or directories do not trigger rebuilds.
      Does not affect the build context.
    live_update: Set of steps for updating a running container
      (see https://docs.tilt.dev/live_update_reference.html)
  """
  # We use --format=docker due to
  # https://github.com/containers/buildah/issues/1589
  # which lots of people are still reporting, even though it's closed :shrug:
  push_cmd = "podman push %s --format=docker $EXPECTED_REF\n" % ref

  custom_build(
    ref=ref,
    command=(
      "set -ex\n" +
      "podman build -t $EXPECTED_REF %s %s\n" +
      push_cmd
    ) % (ref, shlex.quote(context)),
    entrypoint=entrypoint,
    ignore=ignore,
    deps=deps,
    live_update=live_update,
    disable_push=True,
    skips_local_docker=True,
  )

def podman_build_with_restart(
  ref, context, entrypoint=None, dockerfile=None, ignore=None, deps=None, live_update=[]
):
  """Use Podman (https://podman.io/) to build images for Tilt. Wrap a custom_build_with_restart so that the last step
    of any live update is to rerun the given entrypoint.
  Args:
    ref: The name of the image to build. Must match the image
      name in the Kubernetes resources you're deploying.
    context: The build context of the binary to build. Expressed as a file path.
    entrypoint: The command to be (re-)executed when the container starts or when a live_update is run.
    dockerfile: The path to the Dockerfile to use for the build.
    deps: Changes to the given files or directories that will trigger rebuilds.
      Defaults to the build context.
    ignore: Changes to the given files or directories do not trigger rebuilds.
      Does not affect the build context.
    live_update: Set of steps for updating a running container
      (see https://docs.tilt.dev/live_update_reference.html)
  """
  # We use --format=docker due to
  # https://github.com/containers/buildah/issues/1589
  # which lots of people are still reporting, even though it's closed :shrug:
  push_cmd = "podman push %s --format=docker $EXPECTED_REF\n" % ref

  custom_build_with_restart(
    ref=ref,
    command=(
      "set -ex\n" +
      "podman build -t $EXPECTED_REF %s %s\n" +
      push_cmd
    ) % (ref, shlex.quote(context)),
    entrypoint=entrypoint,
    ignore=ignore,
    deps=deps,
    live_update=live_update,
    disable_push=True,
    skips_local_docker=True,
  )

def container_build(
  ref, context, entrypoint=None, dockerfile=None, ignore=None, deps=None, live_update=[]
):
  """Use Apple's container CLI (https://github.com/apple/container) to build images for Tilt
  and load them into the local Kubernetes cluster.
  Args:
    ref: The name of the image to build. Must match the image
      name in the Kubernetes resources you're deploying.
    context: The build context of the binary to build. Expressed as a file path.
    entrypoint: The command to be (re-)executed when the container starts or when a live_update is run.
    dockerfile: The path to the Dockerfile to use for the build.
    deps: Changes to the given files or directories that will trigger rebuilds.
      Defaults to the build context.
    ignore: Changes to the given files or directories do not trigger rebuilds.
      Does not affect the build context.
    live_update: Set of steps for updating a running container
      (see https://docs.tilt.dev/live_update_reference.html)
  """
  push_cmd = "container k8s load-image $EXPECTED_REF"

  custom_build(
    ref=ref,
    command=(
      "set -ex\n" +
      "container build -t $EXPECTED_REF -f %s %s\n" +
      push_cmd
    ) % (dockerfile, shlex.quote(context)),
    entrypoint=entrypoint,
    ignore=ignore,
    deps=deps,
    live_update=live_update,
    disable_push=True,
    skips_local_docker=True,
  )

default_registry('ghcr.io/vasu1124')
allow_k8s_contexts(['colima', 'Default', 'desktop', 'docker-desktop', 'kind-kind', 'rancher-desktop', 'k8s-dev'])

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

# --- Configuration & Tool Multiplexing ---
config.define_string_list('to-run', args=True, usage='Resources to enable')
config.define_string('tool', usage='Container tool to use: "container" (Apple container), "docker", or "podman"')
config.define_string('cluster', usage='Kubernetes cluster name for container k8s (default: k8s-dev)')
cfg = config.parse()

if cfg.get('to-run'):
  config.set_enabled_resources(cfg.get('to-run'))

explicit_tool = cfg.get('tool') or os.getenv('CONTAINER_TOOL') or os.getenv('TILT_CONTAINER_TOOL')
cluster_name = cfg.get('cluster') or os.getenv('K8S_CLUSTER', 'k8s-dev')
current_ctx = k8s_context()

def _check_tool(cmd):
  return bool(local_output(cmd + ' >/dev/null 2>&1 && echo 1 || true', quiet=True))

if explicit_tool:
  selected_tool = explicit_tool.lower().strip()
  print("[Tilt] Using '%s' tool for container builds (configured via flag/env)" % selected_tool)
else:
  # Auto-detection:
  # 1. If Apple container is running and reachable, use container
  if _check_tool('container system status'):
    selected_tool = 'container'
  # 2. If Docker daemon is running and reachable, use docker
  elif _check_tool('docker info'):
    selected_tool = 'docker'
  # 3. If Podman is running and reachable, use podman
  elif _check_tool('podman info'):
    selected_tool = 'podman'
  else:
    selected_tool = 'docker'
  print("[Tilt] Auto-detected '%s' tool for container builds (context: %s)" % (selected_tool, current_ctx))

if selected_tool == 'container':
  container_build(
    'ghcr.io/vasu1124/introspect',
    '.',
    entrypoint=['/introspect-linux'],
    dockerfile='docker/Dockerfile.alpine',
    deps=[
      './introspect-linux',
      './css', 
      './tmpl',
    ],
    live_update=[
      sync('./css', '/css'),
      sync('./tmpl', '/tmpl'),
    ],
  )
elif selected_tool == 'podman':
  podman_build(
    'ghcr.io/vasu1124/introspect',
    '.',
    entrypoint=['/introspect-linux'],
    dockerfile='docker/Dockerfile.alpine',
    deps=[
      './introspect-linux',
      './css', 
      './tmpl',
    ],
    live_update=[
      sync('./css', '/css'),
      sync('./tmpl', '/tmpl'),
    ],
  )
else:
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
#  'mongodb:persistentvolumeclaim',
  'mongodb-secret:secret'],
  labels=['introspect']
)
k8s_resource(workload='etcd', objects=[
  'etcd:secret'],
  labels=['introspect']
)
k8s_resource(workload='valkey', objects=[
#  'valkey:persistentvolumeclaim',
  'valkey-secret:secret'],
  labels=['introspect']
)

#v1alpha1.extension_repo(name='tilt-extensions', url='https://github.com/tilt-dev/tilt-extensions')
#v1alpha1.extension(
#  name='ngrok', 
#  repo_name='tilt-extensions', 
#  repo_path='ngrok',
#)

