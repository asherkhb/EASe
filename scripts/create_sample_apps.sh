#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="${1:-./tmp/ease_apps}"

mkdir -p \
  "$TARGET_DIR/analytics-insights" \
  "$TARGET_DIR/strategy-overview" \
  "$TARGET_DIR/finance/forecast-lab" \
  "$TARGET_DIR/finance/cost-guard" \
  "$TARGET_DIR/engineering/security-hub" \
  "$TARGET_DIR/engineering/monitoring/infrastructure" \
  "$TARGET_DIR/engineering/monitoring/application-performance" \
  "$TARGET_DIR/engineering/database/dev" \
  "$TARGET_DIR/engineering/database/prod"

cat > "$TARGET_DIR/analytics-insights/spec.yml" <<'SPEC'
name: Analytics Insights
description: Executive KPI overview with week-over-week rollups and anomaly callouts.
SPEC

cat > "$TARGET_DIR/strategy-overview/spec.yml" <<'SPEC'
name: Strategy Overview
description: Portfolio progress tracking with quarterly goals and risk signals.
SPEC

cat > "$TARGET_DIR/finance/forecast-lab/spec.yml" <<'SPEC'
name: Forecast Lab
description: Revenue projections, runway scenarios, and variance analysis.
SPEC

cat > "$TARGET_DIR/finance/cost-guard/spec.yml" <<'SPEC'
name: Cost Guard
description: Spend governance, vendor consolidation, and savings opportunities.
SPEC

cat > "$TARGET_DIR/engineering/security-hub/spec.yml" <<'SPEC'
name: Security Hub
description: Threat alerts, audit trails, and compliance posture checks.
SPEC

cat > "$TARGET_DIR/engineering/monitoring/infrastructure/spec.yml" <<'SPEC'
name: Infrastructure
description: Cluster health, capacity forecasting, and availability targets.
SPEC

cat > "$TARGET_DIR/engineering/monitoring/application-performance/spec.yml" <<'SPEC'
name: Application Performance
description: Service latency, error budgets, and transaction tracing.
SPEC

cat > "$TARGET_DIR/engineering/database/dev/spec.yml" <<'SPEC'
name: Dev
description: Development database activity, migrations, and query diagnostics.
SPEC

cat > "$TARGET_DIR/engineering/database/prod/spec.yml" <<'SPEC'
name: Prod
description: Production database uptime, replication, and slow query watchlists.
SPEC

echo "Sample apps created at: $TARGET_DIR"
