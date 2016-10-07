job "volumes" {

	group "v1" {
		task "storage" {
			image = "mystorage:latest"
			volumes = ["/data"]
		}
		task "backup" {
			volumes-from = "storage"
			type = "oneshot"
			image = "mybackup:latest"
			timer = "hourly"
		}
	}

	task "v2" {
		image = "v2:latest"
		volumes = ["/var/run/docker.sock:/tmp/docker.sock", "/etc:/etc"]
	}

	task "v3" {
		image = "v3:latest"
		volumes = ["/var/run/docker.sock:/tmp/docker.sock", "instance:/var/lib/data"]
	}

	task "v4global" {
		global = true
		image = "v4:latest"
		volumes = ["/var/run/docker.sock:/tmp/docker.sock", "instance:/var/lib/data"]
	}
}
