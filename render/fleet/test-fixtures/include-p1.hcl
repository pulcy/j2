group "p1" {
	task "redis" {
        image = "pulcy/ha-redis"
        ports = ["{{private_ipv4}}::6379"]
        args = [
            "--etcd-url=http://{{private_ipv4}}:4001/pulcy/service/${job}-${group}-master/the:master:6379",
            "--container-name ${container}",
            "--docker-url unix:///var/run/docker.sock",
            "--redis-appendonly"
        ]
        volumes = [
            "/var/run/docker.sock:/var/run/docker.sock",
            "${task.volume}/data:/data"
        ]
    }
}
