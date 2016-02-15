package daemon

import (
	// Importing packages here only to make sure their init gets called and
	// therefore they register themselves to the logdriver factory.
	_ "github.com/sara-nl/docker-1.9.1/daemon/logger/awslogs"
	_ "github.com/sara-nl/docker-1.9.1/daemon/logger/fluentd"
	_ "github.com/sara-nl/docker-1.9.1/daemon/logger/gelf"
	_ "github.com/sara-nl/docker-1.9.1/daemon/logger/journald"
	_ "github.com/sara-nl/docker-1.9.1/daemon/logger/jsonfilelog"
	_ "github.com/sara-nl/docker-1.9.1/daemon/logger/syslog"
)
