import os
import re
from pathlib import Path

base_dir = Path(r"c:\Users\ANDREYO\Desktop\PROJECTS\Pos-vertical")

mappings = {
    "better-pos.md": "research/2026-05-11-gaps-vs-competidores.md",
    "finds-betters.md": "research/2026-05-11-features-pedidas-por-tendero.md",
    "find-pos.md": "research/2026-05-verticales-y-rentabilidad.md",
    "better-finds-pos.md": "research/2026-09-08-investigacion-mercado.md",
    "facElect-finds.md": "research/2026-05-dian-proveedores.md",
    "DESIGN-Mesas-Pedidos-KDS.md": "design/2026-09-20-mesas-pedidos-kds.md",
    "fix-gaps.md": "reference/facturacion-electronica.md",
    "ARQUITECTURA.md": "archive/2026-04-17-arquitectura.md",
    "PLAN.md": "archive/2026-05-plan-mvp.md",
    "PRD.md": "archive/2026-05-prd-mvp.md",
    "mvp-context.md": "archive/2026-05-mvp-context.md",
    "BACKLOG-Development-Workflow.md": "workflow.md"
}

def process_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    original_content = content
    
    for old_name, new_name in mappings.items():
        target_abs = base_dir / "project-docs" / new_name
        rel_path = os.path.relpath(target_abs, file_path.parent).replace('\\', '/')
        
        # In CLAUDE.md, they are often referred to as project-docs/OLD_NAME or just OLD_NAME.
        if file_path.name == "CLAUDE.md":
            # If it already has project-docs/
            content = re.sub(r'project-docs/' + re.escape(old_name), f"project-docs/{new_name}", content)
            # If it's just OLD_NAME
            content = re.sub(r'(?<!project-docs/)(?<!/)\b' + re.escape(old_name) + r'\b', new_name, content)
        else:
            # Handle standard references in markdown
            # e.g., [text](OLD_NAME), `OLD_NAME`, OLD_NAME:123
            
            # For README.md, we don't want to replace the left side of the table where it lists the old name.
            # But the user said: "Agrega una columna "Nombre anterior" a las tablas donde aparezcan archivos renombrados"
            # I will manually fix README.md table later. Let's just do the link/path replacements.
            
            # Replace project-docs/OLD_NAME with rel_path (if any exist like this)
            content = re.sub(r'project-docs/' + re.escape(old_name), rel_path, content)
            
            # Replace ../OLD_NAME
            content = re.sub(r'\.\./' + re.escape(old_name), rel_path, content)
            
            # Replace OLD_NAME exactly (word boundary to avoid partial matches)
            # Make sure we don't mess up if it's already part of a path
            content = re.sub(r'(?<!/)(?<!-)\b' + re.escape(old_name) + r'\b', rel_path, content)

    if content != original_content:
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f"Updated {file_path}")

md_files = list((base_dir / "project-docs").rglob("*.md")) + [base_dir / "CLAUDE.md"]

for file in md_files:
    process_file(file)

print("Done")
