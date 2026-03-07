#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

python3 - <<'PY'
from pathlib import Path
import sys

root = Path('/workspace/KubeNAS')
errors = []

def parse_docs(text: str):
    docs=[]
    cur=[]
    for line in text.splitlines():
        if line.strip()=='---':
            if cur:
                docs.append('\n'.join(cur).strip())
                cur=[]
            continue
        cur.append(line)
    if cur:
        docs.append('\n'.join(cur).strip())
    return [d for d in docs if d and not d.strip().startswith('#')]

for path in sorted(root.rglob('*.yaml')):
    if '.git/' in str(path):
        continue
    text=path.read_text(encoding='utf-8')
    docs=parse_docs(text)
    for idx,doc in enumerate(docs,1):
        has_api='apiVersion:' in doc
        has_kind='kind:' in doc
        if not has_api and not has_kind:
            continue
        if not has_api or not has_kind:
            errors.append(f"{path.relative_to(root)} doc#{idx}: missing apiVersion or kind")
            continue
        if 'kind: Deployment' in doc or 'kind: DaemonSet' in doc:
            if 'selector:' not in doc or 'matchLabels:' not in doc or 'template:' not in doc:
                errors.append(f"{path.relative_to(root)} doc#{idx}: missing selector/template structure")

if errors:
    print('Manifest validation failed:')
    print('\n'.join(f'- {e}' for e in errors))
    sys.exit(1)

print('Manifest validation (offline structural checks) passed.')
PY
