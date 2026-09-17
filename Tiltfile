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

def container_build(
    ref, context, dockerfile=None, cluster_name='k8s-dev', push=False, ignore=None, extra_flags=None, deps=None, live_update=[], push_extra_flags=None
):
  """Use Apple's container CLI (https://github.com/apple/container) to build images for Tilt
  and load them into the local Kubernetes cluster.
  Args:
    ref: The name of the image to build. Must match the image
      name in the Kubernetes resources you're deploying.
    context: The build context of the binary to build. Expressed as a file path.
    dockerfile: Path to Dockerfile.
    cluster_name: Name of the local k8s cluster managed by container k8s (defaults to 'k8s-dev').
    push: Set to True if pushing to a remote registry instead of loading into the local cluster.
    deps: Changes to the given files or directories that will trigger rebuilds.
      Defaults to the build context.
    ignore: Changes to the given files or directories do not trigger rebuilds.
    extra_flags: Extra flags to pass to container build. Expressed as an argv-style array.
    push_extra_flags: Extra flags to pass to container image push. Expressed as an argv-style array.
    live_update: Set of steps for updating a running container
      (see https://docs.tilt.dev/live_update_reference.html)
  """
  deps = deps or [context]
  extra_flags = extra_flags or []
  push_extra_flags = push_extra_flags or []

  df_flag = ("-f %s" % shlex.quote(dockerfile)) if dockerfile else ""
  extra_flags_str = ' '.join([shlex.quote(f) for f in extra_flags])
  push_extra_flags_str = ' '.join([shlex.quote(f) for f in push_extra_flags])
  flags = ' '.join([f for f in [df_flag, extra_flags_str] if f])
  if flags:
    flags = " " + flags

  if push:
    load_cmd = "container image push %s $EXPECTED_REF\n" % push_extra_flags_str
  else:
    load_cmd = (
      "if container k8s list 2>/dev/null | grep -q %s; then\n" +
      "  container k8s load-image --name %s $EXPECTED_REF\n" +
      "elif command -v kind >/dev/null 2>&1; then\n" +
      "  kind load docker-image $EXPECTED_REF --name %s\n" +
      "else\n" +
      "  container k8s load-image --name %s $EXPECTED_REF\n" +
      "fi\n"
    ) % (shlex.quote(cluster_name), shlex.quote(cluster_name), shlex.quote(cluster_name), shlex.quote(cluster_name))

  custom_build(
    ref=ref,
    command=(
      "set -ex\n" +
      "container build -t $EXPECTED_REF%s %s\n" +
      load_cmd
    ) % (flags, shlex.quote(context)),
    ignore=ignore,
    deps=deps,
    live_update=live_update,
    skips_local_docker=True,
  )

apple_container_build = container_build

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
  if _check_tool('which container'):
    selected_tool = 'container'
  # 2. If Docker daemon is running and reachable, use docker
  elif _check_tool('which docker'):
    selected_tool = 'docker'
  # 3. If Podman is running and reachable, use podman
  elif _check_tool('which podman'):
    selected_tool = 'podman'
  else:
    selected_tool = 'docker'
  print("[Tilt] Auto-detected '%s' tool for container builds (context: %s)" % (selected_tool, current_ctx))

if selected_tool == 'container':
  container_build(
    'ghcr.io/vasu1124/introspect',
    '.',
    dockerfile='docker/Dockerfile.alpine',
    cluster_name=cluster_name,
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
    extra_flags=['-f', 'docker/Dockerfile.alpine'],
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

