# About
This is a bit of a rework and extending functionalities of the existing consul-watchdog cli tool.

New features:

- aws host discovery
- consul host discovery
- list services
- list services in specific "any unknown passing warning critical"
- list node status
- list target host/s
- deregister service/s in specific state "any unknown passing warning critical"

# AWS auth
The best way to configure credentials on a development machine is to use the ~/.aws/credentials file, which might look like:

```
[default]
aws_access_key_id = AKID1234567890
aws_secret_access_key = MY-SECRET-KEY
```
Alternatively, you can set the following environment variables:

```
AWS_ACCESS_KEY_ID=AKID1234567890
AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
```

# Usage
### Cli help
```
Usage of ./consul-cleaner:

  -ar, --aws-region=eu-west-1         AWS Region. Default: eu-west-1.
  -d, --dryrun                        Dryrun
  -drsrv, --deregister-service        Deregister service. Use it with --service-state.
  -fl, --force-leave-node             Force leave consul node. Use it with --node-status-code.
  -hd, --host-discovery=aws           Host discovery. 'consul' or 'aws' or 'stdin'.
  -lasrv, --list-all-services         List of all services.
  -lchk, --list-checks                List checks.
  -lns, --list-node-status            List nodes status.
  -lsrvis, --list-service-in-state    List of services in specific state. Use it with --service-state.
  -lth, --list-target-hosts           List target hosts.
  -nsc, --node-status-code=4          Consul node status code. Default: 4.
  -p, --port=8500                     Consul members endpoint port. Default: 8500.
  -ss, --service-state=critical       State of the service you wish to deregister. Default: critical.
  -tv, --tag-value=[]                 AWS tag and value. Usage '-tv tag:value'. It is repeatable.
  -u, --url=localhost                 Consul member endpoint. Default: localhost.
  -v, --version                       Consul-cleaner Version.
```


###Example usage
#### passing host/s from stdin
```
echo "11.88.62.55 13.55.22.3" | consul-cleaner -hd stdin -lasrv
```

#### using the consul host discovery
```
consul-cleaner -hd consul -u whatever -p 80 -lasrv
```

#### using the aws host discovery
```
consul-cleaner -hd aws -ar eu-west-1 -t EnvId -tv whatever -lasrv
```

# Build the tool
You need to have go installed on the system where you wish to compile it.
For more information in regards go installation please check https://golang.org/doc/install

```
go get github.com/notonthehighstreet/consul-cleaner
go install github.com/notonthehighstreet/consul-cleaner

```

If the git repo requires ssh key auth you might want to set the global git config to over write the https protocol with the git one. If this is the case please run the following line.

```
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

After running go install if your $GOPATH is set correctly you should find the binary in your $GOPATH/bin folder.

# Docker
Before you begin you need docker and docker-compose installed. For more information please check https://www.docker.com.


If you'd like to create a docker container for the tool please run the following.

The process has 2 stages:

- 1st creates a container to download and compile the source code.
- 2nd creates the actual container which will host the consul-cleaner.

```
git clone https://github.com/notonthehighstreet/consul-cleaner
cd consul-cleaner
docker-compose run builder
docker build ../consul-cleaner
```

#Versioning

Versioning is baked into the build process.

If you pass in the **main.version** variable during the go install process the tool will return the set value if you run **consul-cleaner -v**

```
go install --ldflags "-X main.version='app version' -extldflags -static -s" -v github.com/notonthehighstreet/consul-cleaner
```

Have fun...
