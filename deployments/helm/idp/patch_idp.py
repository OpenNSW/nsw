#!/usr/bin/env python3
"""
Patch IDP Thunder Helm-rendered manifests for OpenShift restricted SCC compliance.

The official Thunder Helm chart (v0.29.0-v0.30.0) hardcodes:
  - runAsUser/runAsGroup/fsGroup: 10001 (violates OpenShift UID range)
  - THUNDER_SKIP_SECURITY: "false" (overrides extraEnv)
  - No HOME env var (needed for writable home under restricted SCC)
  - Ignores global.volumeMounts (signing keys never mounted)

This script fixes all of the above in the rendered YAML before applying.
"""

import yaml
import sys
import copy

# Namespace UID range: 1001570000/10000
TARGET_UID = 1001570000
TARGET_GID = 1001570000


class BlockScalarDumper(yaml.SafeDumper):
    """Custom YAML dumper that uses block scalar (|) for multi-line strings.
    
    This is critical because Thunder's Go binary parses the deployment.yaml
    from the ConfigMap and chokes on PyYAML's default double-quoted format
    which uses escaped \\n characters instead of real newlines.
    """
    pass


def _str_representer(dumper, data):
    if '\n' in data:
        # Use block scalar style (|) for multi-line strings
        return dumper.represent_scalar('tag:yaml.org,2002:str', data, style='|')
    return dumper.represent_scalar('tag:yaml.org,2002:str', data)


BlockScalarDumper.add_representer(str, _str_representer)

def patch_security_context(obj):
    """Recursively fix all runAsUser/runAsGroup/fsGroup set to 10001."""
    if isinstance(obj, dict):
        for k in list(obj.keys()):
            if k in ('runAsUser', 'runAsGroup', 'fsGroup') and obj[k] == 10001:
                if k == 'fsGroup':
                    obj[k] = TARGET_GID
                elif k == 'runAsUser':
                    obj[k] = TARGET_UID
                elif k == 'runAsGroup':
                    obj[k] = TARGET_GID
            else:
                patch_security_context(obj[k])
    elif isinstance(obj, list):
        for item in obj:
            patch_security_context(item)


def patch_env(container):
    """Set THUNDER_SKIP_SECURITY=true and add HOME=/tmp."""
    env = container.get('env', [])
    
    # Fix THUNDER_SKIP_SECURITY
    found_skip = False
    for e in env:
        if e.get('name') == 'THUNDER_SKIP_SECURITY':
            e['value'] = 'true'
            found_skip = True
    if not found_skip:
        env.append({'name': 'THUNDER_SKIP_SECURITY', 'value': 'true'})
    
    # Add HOME=/tmp if missing
    if not any(e.get('name') == 'HOME' for e in env):
        env.append({'name': 'HOME', 'value': '/tmp'})
    
    container['env'] = env


def patch_volumes(pod_spec):
    """Inject thunder-signing-keys volume if not already present."""
    volumes = pod_spec.get('volumes', [])
    if not any(v.get('name') == 'thunder-signing-keys-volume' for v in volumes):
        volumes.append({
            'name': 'thunder-signing-keys-volume',
            'secret': {
                'secretName': 'thunder-signing-keys',
                'defaultMode': 0o444
            }
        })
    pod_spec['volumes'] = volumes


def patch_volume_mounts(container):
    """Inject signing key volume mounts if not already present."""
    mounts = container.get('volumeMounts', [])
    mount_paths = {vm.get('mountPath') for vm in mounts}
    
    needed_mounts = [
        {
            'name': 'thunder-signing-keys-volume',
            'mountPath': '/opt/thunder/repository/certs/signing.cert',
            'subPath': 'signing.cert',
            'readOnly': True
        },
        {
            'name': 'thunder-signing-keys-volume',
            'mountPath': '/opt/thunder/repository/certs/signing.key',
            'subPath': 'signing.key',
            'readOnly': True
        },
        {
            'name': 'thunder-signing-keys-volume',
            'mountPath': '/opt/thunder/repository/certs/crypto.key',
            'subPath': 'crypto.key',
            'readOnly': True
        },
    ]
    
    for m in needed_mounts:
        if m['mountPath'] not in mount_paths:
            mounts.append(m)
    
    container['volumeMounts'] = mounts


def patch_manifest(file_path):
    with open(file_path, 'r') as f:
        docs = list(yaml.safe_load_all(f))

    patched = []
    for doc in docs:
        if not doc:
            continue

        kind = doc.get('kind', '')
        name = doc.get('metadata', {}).get('name', '')

        # Fix 1: Patch all security context UIDs/GIDs everywhere
        patch_security_context(doc)

        # Fix 2-4: Patch Deployments and Jobs
        if kind in ('Deployment', 'Job') and 'idp-thunder' in name:
            pod_spec = doc['spec']['template']['spec']
            
            # Inject volumes
            patch_volumes(pod_spec)
            
            # Patch each container
            for container in pod_spec.get('containers', []):
                patch_env(container)
                patch_volume_mounts(container)
            
            # Also patch init containers
            for container in pod_spec.get('initContainers', []):
                patch_volume_mounts(container)

            print(f"  ✓ Patched {kind}: {name}")

        # Fix 5: Hardcoded Configuration Bypass in ConfigMap
        if kind == 'ConfigMap' and name == 'idp-thunder-config-map':
            if 'data' in doc and 'deployment.yaml' in doc['data']:
                content = doc['data']['deployment.yaml']
                # Replace password tokens with "postgres"
                content = content.replace('\"{{.DB_CONFIG_PASSWORD}}\"', '\"postgres\"')
                content = content.replace('\"{{.DB_RUNTIME_PASSWORD}}\"', '\"postgres\"')
                content = content.replace('\"{{.DB_USER_PASSWORD}}\"', '\"postgres\"')
                # Also replace unquoted ones just in case
                content = content.replace('{{.DB_CONFIG_PASSWORD}}', 'postgres')
                content = content.replace('{{.DB_RUNTIME_PASSWORD}}', 'postgres')
                content = content.replace('{{.DB_USER_PASSWORD}}', 'postgres')
                
                doc['data']['deployment.yaml'] = content
                print(f"  ✓ Patched ConfigMap: {name} (Hardcoded Bypass)")

        patched.append(doc)

    with open(file_path, 'w') as f:
        yaml.dump_all(patched, f, Dumper=BlockScalarDumper, default_flow_style=False)
    
    print(f"  ✓ Wrote {len(patched)} documents to {file_path}")


if __name__ == '__main__':
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <manifest.yaml>")
        sys.exit(1)
    patch_manifest(sys.argv[1])
