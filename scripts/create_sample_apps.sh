#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="${1:-./tmp/ease_apps}"

mkdir -p \
  "$TARGET_DIR/analytics-insights" \
  "$TARGET_DIR/monitoring/infra-core" \
  "$TARGET_DIR/monitoring/security-hub" \
  "$TARGET_DIR/finance/forecast-lab" \
  "$TARGET_DIR/ml/feature-store" \
  "$TARGET_DIR/customer/growth-lab" \
  "$TARGET_DIR/operations/supply-line" \
  "$TARGET_DIR/marketing/brand-pulse"

cat > "$TARGET_DIR/analytics-insights/spec.yml" <<'SPEC'
name: Analytics Insights
description: Executive KPI overview with week-over-week rollups and anomaly callouts.
SPEC

cat > "$TARGET_DIR/monitoring/infra-core/spec.yml" <<'SPEC'
name: Infra Core Monitor
description: Live infrastructure health, latency, and error budgets across clusters.
SPEC

cat > "$TARGET_DIR/monitoring/security-hub/spec.yml" <<'SPEC'
name: Security Hub
description: Threat alerts, audit trails, and compliance posture checks.
SPEC

cat > "$TARGET_DIR/finance/forecast-lab/spec.yml" <<'SPEC'
name: Forecast Lab
description: Revenue projections, runway scenarios, and variance analysis.
SPEC

cat > "$TARGET_DIR/ml/feature-store/spec.yml" <<'SPEC'
name: Feature Store Explorer
description: Browse feature definitions, freshness, and training coverage.
SPEC

cat > "$TARGET_DIR/customer/growth-lab/spec.yml" <<'SPEC'
name: Growth Lab
description: Cohort retention, activation funnels, and experiment outcomes.
SPEC

cat > "$TARGET_DIR/operations/supply-line/spec.yml" <<'SPEC'
name: Supply Line Console
description: Shipment tracking, vendor SLAs, and inventory risk alerts.
SPEC

cat > "$TARGET_DIR/marketing/brand-pulse/spec.yml" <<'SPEC'
name: Brand Pulse
description: Campaign performance, channel attribution, and sentiment trends.
SPEC

echo "Sample apps created at: $TARGET_DIR"
