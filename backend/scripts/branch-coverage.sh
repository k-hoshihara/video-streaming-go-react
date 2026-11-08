#!/bin/sh
# gobco で分岐網羅（C1）を測り、octocov のカスタムメトリクス形式の JSON を出力する。
# go test -cover は命令網羅（C0）しか測れないため、その補完として使う。
#
# 使い方（backend/ で実行）: scripts/branch-coverage.sh [出力先 JSON]
# 通っていない分岐は標準出力に、GitHub Actions 上ではジョブサマリーにも出力する。
set -eu

out=${1:-branch-coverage.json}
module=$(go list -m)
covered=0
total=0
uncovered=""

# 自動生成コード（internal/oapi）とテスト用のヘルパー（internal/testdb）は対象外
for pkg in $(go list ./... | grep -v -E '/internal/(oapi|testdb)$'); do
  dir=${pkg#"$module"/}
  if ! result=$(cd "$dir" && go tool gobco -branch . 2>&1); then
    echo "$result"
    echo "gobco failed in $dir" >&2
    exit 1
  fi

  # 例: "Branch coverage: 7/8"
  counts=$(echo "$result" | sed -n 's/^Branch coverage: \([0-9]*\)\/\([0-9]*\)$/\1 \2/p')
  covered=$((covered + ${counts% *}))
  total=$((total + ${counts#* }))

  # 例: "server.go:27:5: condition "err != nil" was 10 times false but never true"
  lines=$(echo "$result" | grep -E '^[^ ]+\.go:[0-9]+:[0-9]+: ' | sed "s|^|$dir/|" || true)
  if [ -n "$lines" ]; then
    uncovered="$uncovered$lines
"
  fi
done

if [ "$total" -eq 0 ]; then
  percent=100
else
  percent=$(awk "BEGIN { printf \"%.1f\", $covered * 100 / $total }")
fi

cat > "$out" <<EOF
{
  "key": "branch_coverage",
  "name": "Branch Coverage (C1, gobco)",
  "metrics": [
    { "key": "coverage", "name": "Coverage", "value": $percent, "unit": "%" },
    { "key": "covered", "name": "Covered", "value": $covered },
    { "key": "total", "name": "Total", "value": $total }
  ]
}
EOF

report="Branch coverage (C1): $covered/$total ($percent%)"
echo "$report"
if [ -n "$uncovered" ]; then
  printf 'Uncovered branches:\n%s' "$uncovered"
fi

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    echo "### $report"
    if [ -n "$uncovered" ]; then
      echo ""
      echo "通っていない分岐:"
      echo ""
      echo '```'
      printf '%s' "$uncovered"
      echo '```'
    fi
  } >> "$GITHUB_STEP_SUMMARY"
fi
