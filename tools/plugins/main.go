package plugins

import (
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_instances"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_log_download"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_logs"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_metrics"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_parameter_groups"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/aws_rds_performance_insights"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/pt_query_digest"
	_ "github.com/nicola-strappazzon/argos/tools/plugins/pt_variable_advisor"
)
