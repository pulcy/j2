# J2

[`j2`](#why-is-it-called-j2) is the pulcy service deployment tool.
It takes a job description as input and generates (fleet) unit files for all tasks in the given job.
The unit files will be pushed onto a CoreOS cluster.

##  Usage

To create/update a [job](#job-specification) on a [cluster](#cluster-specification), run:

```
j2 run -j <jobpath> -c <clusterpath> [-o <optionspath]
```

To completely remove a job from a cluster, run:

```
j2 destroy -j <jobpath> -c <clusterpath>
```

## Job specification

A job is a logical group of services.

A job contains one or more task groups.
A task group is a group of tasks that are scheduled on the same machine.
A task specifies a single container.

A very basic job looks like this:

```
job "basic" {
    task "someservice" {
        image = "myimage"
    }
}
```

Jobs are specified in [HCL format](https://github.com/hashicorp/hcl).
A job always has a name and only one job can be specified per file.

Objects in a job can be `task`, `group` and `constraint`.
If you add a `task` directly to a `job`, it will be automatically wrapped in a `task-group`.

The following keys can be specified on a `job`.

- `id` - `id` is used to give a job a unique identifier which is used for authentication when fetching secrets.
- `constraint` - See [Constraints](#constraints)

### Tasks

A `task` is an object that specifies something that will be executed in a container.

The following keys can be specified on a `task`.

- `args` - Contains the command line arguments passed during the start of the container.
- `environment` - Contains zero of more environment variables passed during the start of the container.
- `image` - Specifies the docker image that will be executed.
- `type` - The type of a task can be "service" (default) or "oneshot".
  Oneshot tasks are supposed to run for a while and then exit.
  Service tasks are supposed to run continuously and will be restarted in case of failure.
- `volumes-from` - Contains the name of zero or more other tasks. The volumes used by these other tasks will be
  mounted in the container of this task.
- `volumes` - Contains a list of zero or more volume mounts for the container.
  Each volumes entry must be a valid docker volume description ("hostpath:containerpath")
- `ports` - Contains a list of port specifications that specify which ports (exposed by the container) will be
  mapped into the port namespace of the machine on which the container is scheduled.
  Each port entry must be a valid docker port specification.
  Note that `ports` are not often used. In most cases you'll use a `frontend` or `private-frontend`.
- `links` - Contains a list of task names that this task will be able to access through their private frontends.
Each name must be a fully qualified task name (job.group.task).
- `capabilities` - Contains a list of Linux capabilities to add to the container. (See `docker run --cap-add`)
- `constraint` - See [Constraints](#constraints)
- `http-check-path` - Contains an HTTP path for the load-balancer to call when checking the status of this task.
- `frontend` - Contains a public load-balancer registration. This configures the load-balancer to forward certain requests from the public network interface(s) of the cluster to this task. See [Frontends](#frontends).
- `private-frontend` - Contains a private load-balancer registration. This configures the load-balancer to forward certain requests from the private network interface(s) of the cluster to this task. See [Frontends](#frontends).
- `secret` - Contains a specification for a secret value to be fetched and mapped into the container. See [Secret](#secrets).
- `log-driver` - Specifies the log-driver to use in docker. This can be "" (default) or "none". If not equal to "none",
the `log-args` or the `docker` settings in the [cluster](#cluster-specification) are used.

#### Frontends

Frontends are used to provide a configuration for the load-balancer.

The following keys can be specified on a public frontend.

- `domain` - The load-balancer will forward requests that match this domain.
- `path-prefix` - The load-balancer will forward requests where the path of the requests starts with this prefix.
- `ssl-cert` - The load-balancer will use an SSL certificate with this filename for connections to this task. If you do not specify an SSL certificate and the load-balancer is configured to use [Let's Encrypt](https://letsencrypt.org) a certificate will be automatically created for the specified `domain`.

The following keys can be specified on a all frontends.

- `port` - The load-balancer will forward requests to this port of the task. If you do not specify a port, it will forward requests to any of the ports exposed by the container.
- `user` - User objects specify password authentication to be used for requests forwarded for this task.
- `weight` - Contains a value [0...100] used to order frontend specifications in the load-balancer. If 2 frontend specifications both match a specific request, the one with the highest weight will be used.

The following keys can be specified on a private frontend.

- `mode` - Specifies the mode the load-balancer will be configured for for this frontend. Mode can be `http` (default) or `tcp`.
Frontends using `tcp` mode will offer TCP over TLS connections, using SNI to identify the correct task to forward the request to.
The connection from the load-balancer to the task will use TCP only.
- `register-instance` - If set, instances of this task will also be registered in the load-balancer under an instance specific
name. This enables access to individual instances, in addition to load-balanced access to all instances of a task.

#### Secrets

Secrets are used to pass sensitive data to tasks in a secure manor.
The sensitive data can be exposed as an environment variable (e.g. passwords) or as a file (e.g. certificates).
Secrets are extracted from a [Vault](https://vaultproject.io).

A secret looks like this:

```
secret "secret/mypassword" {
    environment = "MYPASSWORD"
}
```

The above secret results in the value of a secret under path "secret/mypassword" to be passed to the task in an environment variable named "MYPASSWORD".

```
secret "secret/mycertificate" {
    file = "/config/mycertificate.pem"
}
```

The above secret results in the value of a secret under path "secret/mycertificate" to be passed to the task in a file mounted on "/config/mycertificate.pem".

The following keys can be specified on a `secret`.

- `field` - Contains the name of the field of the secret. If no field is specified, the `value` field is fetched.
- `environment` - Contains the name of the environment variable that will be passed into the container.
- `file` - Contains the full path of the file that will be mounted into the container.

You must specify an `environment` or a `file`, not both.

### Task groups

A `group` is an object that groups one or more tasks such that they are always scheduled on the same machine.
You use the `count` key to specify how many instances of a task-group you want to run. Each instance of a task-group
contains a container for all tasks in that task-group. Multiple instances of a task-group are not guaranteed to run on
the same machine. In fact you often want different instances to run on different machines for high availability.

The following keys can be specified on a `group`.

- `count` - Specifies how many instances of the task-group that should be created.
- `global` - If set to true, this task-group will create one instance for every machine in the cluster.
- `constraint` - See [Constraints](#constraints)

### Constraints

With constraints you can control on which machines tasks can be scheduled.
Constraints can be specified on `job`, `group` and `task` level. Constrains on a deeper level overwrite constraints
on high levels with the same `attribute`.

Here's an example of a constraint that forced a task to be scheduled on a machine that has `region=eu-west` in
its metadata.

```
constraint {
    attribute = "meta.region"
    value = "eu-west"
}
```

The following keys can be specified on a `constraint`.

- `attribute` - One of the attributes you can filter on. See [Attributes](#attributes) below.
- `value` - The value for this attribute.

#### Attributes

The following attributes can be used in constraints.

- `meta.<key>` - Refers to a key used in the metadata of a machine.
- `node.id` - Refers to the `machine-id` of a machine.

## Cluster specification

A cluster file specifies those attributes of a cluster that are relevant for deploying jobs on it.

A typical cluster file looks like:

```
cluster "production" {
    domain = "example.com"
    stack = "production"
    instance-count = 3

    docker {
        log-args = ["--log-driver=fluentd", "--log-opt fluentd-address=127.0.0.1:24284"]
    }

    fleet {
        after = [
            "gluon.service",
            "loggly-fluentd-fluentd-mn.service"
        ]
        wants = "loggly-fluentd-fluentd-mn.service"
    }

    default-options {
        "force-ssl" = "true"
    }
}
```

The following keys can be specified on a `cluster`.

- `domain` - The domain name of the cluster
- `stack` - The name of the stack to deploy in. The combination of `stack` + `domain` forms the DNS name that is
used to deploy units to.
- `instance-count` - The number of machines in the cluster. If this number is higher than the `count` of a task-group,
different instances of that task-group will be forced on different machines.
- `docker` - A set of options applied to all docker commands generated for jobs on this cluster.
- `docker.log-args` - Docker command line arguments related to logging. Uses when `log-driver` is not equal to `none`.
- `fleet` - A set of options applied to all fleet units generated for jobs on this cluster.
- `fleet.after` - A list of unit names to add to the [After](https://www.freedesktop.org/software/systemd/man/systemd.unit.html#After=) setting of each unit.
This list is filtered such that units originating from the same job are excluded.
- `fleet.wants` - A list of unit names to add to the [Wants](https://www.freedesktop.org/software/systemd/man/systemd.unit.html#Wants=) setting of each unit.
This list is filtered such that units originating from the same job are excluded.
- `fleet.requires` - A list of unit names to add to the [Requires](https://www.freedesktop.org/software/systemd/man/systemd.unit.html#Requires=) setting of each unit.
This list is filtered such that units originating from the same job are excluded.

## Why is it called J2?

This tool is named after the famous [J-2](https://en.wikipedia.org/wiki/J-2_%28rocket_engine%29) rocket engine that helped bring man to the moon. It was a predecessor for the RS-25 rocket engine that powered the Space Shuttle and even today it is an inspiration for the J-2X engine intended for NASA's Space Launch System.
