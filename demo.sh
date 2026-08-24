#!/usr/bin/env bash
set -e

# Build SafeShell binary
go build -o safeshell .
export PATH="$PWD:$PATH"

echo "=== 1. Clean Environment ==="
rm -rf workspace snapshots tmp evidence

echo -e "\n=== 2. SafeShell Init ==="
safeshell init

echo -e "\n=== 3. Execute 'safeshell run mkdir docs' ==="
safeshell run mkdir docs

echo -e "\n=== 4. Execute 'safeshell run touch notes.txt' ==="
safeshell run touch notes.txt

echo -e "\n=== 5. Inspect Workspace State ==="
find workspace

echo -e "\n=== 6. Rollback Latest Command ==="
safeshell rollback latest

echo -e "\n=== 7. Inspect Workspace State After Rollback ==="
find workspace

echo -e "\n=== 8. View Latest Evidence ==="
safeshell evidence latest

echo -e "\n=== 9. Test Dangerous Command Rejection ('rm -rf /') ==="
set +e
safeshell run rm -rf /
EXIT_CODE_1=$?
set -e
if [ $EXIT_CODE_1 -ne 0 ]; then
    echo "Successfully rejected dangerous command (exit code $EXIT_CODE_1)."
else
    echo "FAILED: Dangerous command was not rejected!"
    exit 1
fi

echo -e "\n=== 10. Test Path Escape Rejection ('mkdir ../../evil') ==="
set +e
safeshell run mkdir ../../evil
EXIT_CODE_2=$?
set -e
if [ $EXIT_CODE_2 -ne 0 ]; then
    echo "Successfully rejected path escape (exit code $EXIT_CODE_2)."
else
    echo "FAILED: Path escape was not rejected!"
    exit 1
fi

echo -e "\n=== Demo Completed Successfully ==="
