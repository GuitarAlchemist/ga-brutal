"""Lightweight integrity, boundary and secret-pattern checks; no runtime or network."""
from pathlib import Path
import ast
import hashlib
import json
import re
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parents[1]
provenance = json.loads((ROOT / 'PROVENANCE.json').read_text(encoding='utf-8'))
modified = []
for row in provenance['files']:
    file = ROOT / row['path']
    if not row['included']:
        assert not file.exists(), f'excluded upstream data appeared: {file}'
        continue
    assert hashlib.sha256(file.read_bytes()).hexdigest() == row['extracted_sha256'], file
    if row['modified_for_extraction']:
        modified.append(row['path'])
assert sorted(modified) == sorted([
    'Experiments/Brutal/Colossus/Program.cs',
    'Experiments/Brutal/Colossus/Speech.cs',
    'Experiments/Brutal/Colossus/Colossus.csproj',
    'Experiments/Brutal/Colossus/Streamline.cs',
    'Experiments/Brutal/README.md',
])
projects = list(ROOT.glob('Experiments/Brutal/*/*.csproj')) + list(ROOT.glob('Experiments/Brutal/*/*.fsproj'))
assert len(projects) == 6
lego = (ROOT / 'Experiments/Brutal/Lego/Lego.fsproj').resolve()
for project in projects:
    tree = ET.parse(project)
    # The only project reference allowed is the snapshot's own Lego library, never GA.
    for reference in tree.findall('.//ProjectReference'):
        target = (project.parent / reference.attrib['Include'].replace('\\', '/')).resolve()
        assert target == lego, (project, target)
    assert not tree.findall('.//Import'), project
    assert (project.parent / 'packages.lock.json').exists()
    for package in tree.findall('.//PackageReference'):
        assert 'fmod' not in package.attrib.get('Include', '').lower()
    for item in tree.findall('.//None'):
        name = item.attrib['Include'].replace('\\', '/')
        if '*' not in name and ';' not in name:
            assert (project.parent / name).exists() or 'Exists(' in item.attrib.get('Condition', ''), name
    lock = json.loads((project.parent / 'packages.lock.json').read_text())
    for framework, packages in lock['dependencies'].items():
        for name, package in packages.items():
            assert package.get('type', '').lower() != 'project' or name.lower() == 'lego', (project, name)
            assert 'fmod' not in name.lower()
assert len(list(ROOT.glob('Experiments/Brutal/Colossus/textures/*.png'))) == 12

blocked = {'.exe', '.dll', '.pdb', '.wav', '.onnx', '.safetensors', '.gguf', '.pt', '.pth', '.bin', '.glb', '.mp4'}
secret_patterns = [
    re.compile(r'gh[pousr]_[A-Za-z0-9]{30,}'),
    re.compile(r'github_pat_[A-Za-z0-9_]{30,}'),
    re.compile(r'sk-(?:proj-|ant-)?[A-Za-z0-9_-]{25,}'),
    re.compile(r'AKIA[0-9A-Z]{16}'),
    re.compile(r'-----BEGIN (?:RSA |OPENSSH |EC )?PRIVATE KEY-----'),
    re.compile(r'https?://[^\s/@:]+:[^\s/@]+@'),
]
files = []
for file in ROOT.rglob('*'):
    rel = file.relative_to(ROOT)
    if any(part in {'.git', 'bin', 'obj', 'validation-logs', '__pycache__'} for part in rel.parts):
        continue
    if not file.is_file():
        continue
    assert not file.is_symlink()
    assert file.suffix.lower() not in blocked, file
    assert not file.name.startswith('.env'), file
    files.append(file)
    if file.suffix.lower() == '.py':
        ast.parse(file.read_text(encoding='utf-8'), filename=str(rel))
    if file.suffix.lower() not in {'.png'}:
        text = file.read_text(encoding='utf-8-sig')
        for pattern in secret_patterns:
            assert not pattern.search(text), f'secret pattern match in {rel}; inspect without displaying the match'

for row in provenance['licence_texts']:
    assert hashlib.sha256((ROOT / row['path']).read_bytes()).hexdigest() == row['sha256']
assert '2021 GuitarAlchemist' in (ROOT / 'LICENSE').read_text(encoding='utf-8')
assert 'BRUTAL License v1.0' in (ROOT / 'LICENSES/BRUTAL/LICENSE.txt').read_text(encoding='utf-8')
assert '2024 Ahwoo Limited' in (ROOT / 'NOTICE.txt').read_text(encoding='utf-8')
assert 'gaia/issues/257' in (ROOT / 'README.md').read_text(encoding='utf-8')
assert '11.0.0-preview.5.26302.115' in (ROOT / 'README.md').read_text(encoding='utf-8')
excluded = sum(not row['included'] for row in provenance['files'])
print(f'PASS: {len(files)} source/licence/document files; 6 projects; 12 textures; all included blobs verified; {excluded} excluded assets absent; no GA references or secret-pattern matches.')
