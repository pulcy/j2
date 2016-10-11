job "proxyweave" {

    task "db" {
        image = "myhttpdatabase"
        ports = ["{{private_ipv4}}::1234"]
        network = "weave"
    }

	task "dbproxy" {
        type = "proxy"
        target = ".db"
        rewrite {
            path-prefix = "/_db/db1/proxy/"
        }
        private-frontend {
            port = 8529
        }
	}

	task "dbclient" {
        image = "myclient"
        args = ["${link_url .dbproxy}"]
	}

	task "dbclientweave" {
        image = "myclient"
        args = ["${link_url .dbproxy}"]
        network = "weave"
	}
}
