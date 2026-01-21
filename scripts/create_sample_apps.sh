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
  "$TARGET_DIR/engineering/monitoring/database/dev" \
  "$TARGET_DIR/engineering/monitoring/database/prod"

# Public apps (no group restrictions)
cat > "$TARGET_DIR/analytics-insights/spec.yml" <<'SPEC'
name: Analytics Insights
description: Executive KPI overview with week-over-week rollups and anomaly callouts.
SPEC

cat > "$TARGET_DIR/strategy-overview/spec.yml" <<'SPEC'
name: Strategy Overview
description: Portfolio progress tracking with quarterly goals and risk signals.
SPEC

# Finance apps (restricted to finance and admin groups)
cat > "$TARGET_DIR/finance/forecast-lab/spec.yml" <<'SPEC'
name: Forecast Lab
description: Revenue projections, runway scenarios, and variance analysis.
groups:
  - finance
  - admin
SPEC

cat > "$TARGET_DIR/finance/cost-guard/spec.yml" <<'SPEC'
name: Cost Guard
description: Spend governance, vendor consolidation, and savings opportunities.
groups:
  - finance
  - admin
SPEC

# Security app (admin only)
cat > "$TARGET_DIR/engineering/security-hub/spec.yml" <<'SPEC'
name: Security Hub
description: Threat alerts, audit trails, and compliance posture checks.
groups:
  - admin
SPEC

# Infrastructure monitoring (engineering and admin)
cat > "$TARGET_DIR/engineering/monitoring/infrastructure/spec.yml" <<'SPEC'
name: Infrastructure
description: Cluster health, capacity forecasting, and availability targets.
groups:
  - engineering
  - admin
SPEC

# Application performance (engineering and developers)
cat > "$TARGET_DIR/engineering/monitoring/application-performance/spec.yml" <<'SPEC'
name: Application Performance
description: Service latency, error budgets, and transaction tracing.
groups:
  - engineering
  - developers
SPEC

# Dev database (developers only)
cat > "$TARGET_DIR/engineering/monitoring/database/dev/spec.yml" <<'SPEC'
name: Dev
description: Development database activity, migrations, and query diagnostics.
groups:
  - developers
SPEC

# Prod database (engineering and admin)
cat > "$TARGET_DIR/engineering/monitoring/database/prod/spec.yml" <<'SPEC'
name: Prod
description: Production database uptime, replication, and slow query watchlists.
groups:
  - engineering
  - admin
SPEC

# Create example users file
cat > "$TARGET_DIR/users.txt" <<'USERS'
# EASe Users File
# Format: username password group1,group2,group3
# Groups are comma-separated and optional

# Admin user - can see everything
admin adminpass admin

# Finance team - can see finance apps and public apps
finance-alice financepass finance
finance-bob financepass finance

# Engineering team - can see engineering apps and public apps
engineer-charlie engpass engineering
engineer-diana engpass engineering

# Developers - can see dev environments and application monitoring
dev-eve devpass developers
dev-frank devpass developers

# Executive - finance and high-level visibility
exec-grace execpass finance,admin

# Readonly user - can only see public apps (no groups)
readonly-user readpass
USERS

echo "=================================================="
echo "Sample apps created at: $TARGET_DIR"
echo ""
echo "App access by group:"
echo "  - Public (no auth required):"
echo "      • Analytics Insights"
echo "      • Strategy Overview"
echo ""
echo "  - finance + admin:"
echo "      • Forecast Lab"
echo "      • Cost Guard"
echo ""
echo "  - admin only:"
echo "      • Security Hub"
echo ""
echo "  - engineering + admin:"
echo "      • Infrastructure"
echo "      • Prod Database"
echo ""
echo "  - engineering + developers:"
echo "      • Application Performance"
echo ""
echo "  - developers only:"
echo "      • Dev Database"
echo ""
echo "=================================================="
echo "Sample users file created at: $TARGET_DIR/users.txt"
echo ""
echo "Available users:"
echo "  admin / adminpass          (groups: admin)"
echo "  finance-alice / financepass (groups: finance)"
echo "  engineer-charlie / engpass  (groups: engineering)"
echo "  dev-eve / devpass           (groups: developers)"
echo "  exec-grace / execpass       (groups: finance,admin)"
echo "  readonly-user / readpass    (groups: none)"
echo ""
echo "To run with authentication:"
echo "  ./ease -apps-dir $TARGET_DIR -auth-backend file -file-auth-users $TARGET_DIR/users.txt"
echo ""
echo "Or without authentication (public mode):"
echo "  ./ease -apps-dir $TARGET_DIR"
echo "=================================================="
