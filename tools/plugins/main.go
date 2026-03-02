package plugins

import (
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_health_events"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_secrets_get"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_secrets_list"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/mysql_databases"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/mysql_tables"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/mysql_global_variables"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_events"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_pending_maintenance"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_read_replicas"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_snapshots"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_instances"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_log_download"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_logs"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_metrics"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_parameter_groups"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_performance_insights"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/pt_query_digest"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/pt_variable_advisor"
)
