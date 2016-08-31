package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/cocooma/awsdiscovery"
	flag "github.com/cocooma/mflag"
	"github.com/docker/docker/opts"
	"github.com/hashicorp/consul/api"
)

var (
	str, url, port, srvState, srvName, awsregion, tag, tagvalue, hostdiscovery, line, version                                                                          string
	listTargetHosts, showMmemberStatus, listChecks, listAllSrv, deregisterSrv, listSrvInState, listNodeStatus, forceLeaveNode, dryRun, validnode, ver, rnptSet, dsnSet bool
	nsc                                                                                                                                                                int
	hosts                                                                                                                                                              []string
	wg                                                                                                                                                                 sync.WaitGroup
	tagsValues                                                                                                                                                         = opts.NewListOpts(nil)
	srvnameNode                                                                                                                                                        = opts.NewListOpts(validateSNisSet)
	srvnameNodePortTag                                                                                                                                                 = opts.NewListOpts(validateSNPTisSet)
)

func validateSNPTisSet(val string) (string, error) {
	if val != "" {
		rnptSet = true
		return val, nil
	}
	return "", fmt.Errorf("%s snpt is not set", val)
}

func validateSNisSet(val string) (string, error) {
	if val != "" {
		dsnSet = true
		return val, nil
	}
	return "", fmt.Errorf("%s sn is not set", val)
}

func connection(uurl, pport string) *api.Client {
	connection, err := api.NewClient(&api.Config{Address: uurl + ":" + pport})
	if err != nil {
		panic(err)
	}
	return connection
}

func consulmembers(consulClient *api.Client) []string {
	var ips []string
	members, _ := consulClient.Agent().Members(false)
	for _, server := range members {
		ips = append(ips, server.Name)
	}
	return ips
}

func awshosts(awsregion string, tagsValues opts.ListOpts) []string {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Whoops some error happened:", e)
		}
	}()
	var allIps []string
	tagsValuesSlice := tagsValues.GetAll()
	session := awsdiscovery.AwsSessIon(awsregion)
	for _, val := range tagsValuesSlice {
		tv := strings.Split(val, ":")
		tag, value := tv[0], tv[1]
		filter := awsdiscovery.AwsFilter(tag, value)
		ips := awsdiscovery.AwsInstancePrivateIP(session, filter)
		allIps = append(allIps, ips...)
	}
	return allIps
}

func readHostsFromStdin() []string {
	var ips []string
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line = scanner.Text()
		lines := strings.Split(line, " ")
		for _, word := range lines {
			ips = append(ips, word)
		}
	}
	return ips
}

func listTargetHost(ips []string) {
	for _, server := range ips {
		fmt.Println(server)
	}
}

func listNodeStat(consulClient *api.Client) {
	members, _ := consulClient.Agent().Members(false)
	for _, server := range members {
		fmt.Printf("%s %v \n", server.Name, server.Status)
	}
}

func listCheck(consulClient *api.Client) {
	checks, _ := consulClient.Agent().Checks()
	for _, check := range checks {
		fmt.Println(check.Node, check.Name, check.Status)
	}
}

func getServiceID(consulClient *api.Client, serviceName string) {
	service, _, _ := consulClient.Catalog().Service(serviceName, "", nil)
	// return service
	for _, srv := range service {
		fmt.Println(srv.ServiceID)
	}
}

func listServices(consulClient *api.Client) {
	services, _, _ := consulClient.Catalog().Services(nil)
	for srv := range services {
		fmt.Println(srv)
	}
}

func listServicesInState(consulConnection *api.Client, serviceCheckStatus string) {
	service := serviceNameServiceID(consulConnection, serviceCheckStatus)
	for serviceName, serviceID := range service {
		fmt.Println(serviceName + " " + serviceID)
	}
}

func serviceNameServiceID(connection *api.Client, serviceCheckStatus string) map[string]string {
	services := map[string]string{}
	serv, _, _ := connection.Health().State(serviceCheckStatus, nil)
	for _, key := range serv {
		services[key.ServiceName] = key.ServiceID
	}
	return services
}

func registerService(consulConnection *api.Client, srvnameNodePortTag opts.ListOpts) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Whoops some error happened:", e)
		}
	}()

	srvnameNodePortTagSlice := srvnameNodePortTag.GetAll()
	for _, val := range srvnameNodePortTagSlice {
		// add service tag if it is not specified
		if strings.Count(val, ":") == 2 {
			val = val + ":"
		}

		sapt := strings.Split(val, ":")
		srvname, addr, prt, tag := sapt[0], sapt[1], sapt[2], sapt[3]

		if srvname != "" || addr != "" {
			port, _ := strconv.Atoi(prt)
			service := &api.AgentService{
				ID:      srvname,
				Service: srvname,
				Port:    port,
				Tags:    []string{tag},
			}
			register := api.CatalogRegistration{
				Node:    addr,
				Service: service,
				Address: addr,
				Check:   nil,
			}
			_, err := consulConnection.Catalog().Register(&register, nil)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Println("There is no service with the given name: - " + srvname + " - available to register!")
		}
	}
}

func deregisterNodes(consulConnection *api.Client, srvnameNode opts.ListOpts) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Whoops some error happened:", e)
		}
	}()

	srvnameNodeSlice := srvnameNode.GetAll()
	for _, val := range srvnameNodeSlice {
		sn := strings.Split(val, ":")
		srvname, node := sn[0], sn[1]
		if srvname != "" || node != "" {
			dereg := api.CatalogDeregistration{
				ServiceID: srvname,
				Node:      node,
				Address:   node,
			}
			deregnode := api.CatalogDeregistration{
				Node: node,
			}
			_, err := consulConnection.Catalog().Deregister(&dereg, nil)
			if err != nil {
				panic(err)
			}
			_, errr := consulConnection.Catalog().Deregister(&deregnode, nil)
			if errr != nil {
				panic(errr)
			}
		} else {
			fmt.Println("There is no service with the given name: - " + srvname + " - available to deregister!")
		}
	}
}

func deregisterServiceInState(consulConnection *api.Client, serviceCheckStatus string) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Whoops some error happened:", e)
		}
	}()
	service := serviceNameServiceID(consulConnection, serviceCheckStatus)
	if len(service) == 0 {
		fmt.Println("There is no service in the given state: - " + serviceCheckStatus + " - available to deregister!")
	} else {
		for serviceName, serviceID := range service {
			switch dryRun {
			case false:
				fmt.Println("SrvName: " + serviceName + "  SrvID: " + serviceID + "  SrvStatus: " + serviceCheckStatus + "  Has been deregistered!")
				err := consulConnection.Agent().ServiceDeregister(serviceID)
				if err != nil {
					panic(err)
				}
			case true:
				fmt.Println("Dryrun!!!   SrvName: " + serviceName + "  SrvID: " + serviceID + "  SrvStatus: " + serviceCheckStatus + "  Has been deregistered!")
			}
		}
	}
}

func forceLeaveBadNode(consulClient *api.Client, nodeStatusCode int, member string) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Whoops some error happened:", e)
		}
	}()
	validnode = false
	members, _ := consulClient.Agent().Members(false)
	for _, server := range members {
		if server.Status == nodeStatusCode {
			validnode = true
			switch dryRun {
			case false:
				fmt.Printf("On member: %s  node: %s  -  force left the cluster!\n", member, server.Name)
				err := consulClient.Agent().ForceLeave(server.Name)
				if err != nil {
					panic(err)
				}
			case true:
				fmt.Printf("Dryrun!!!   On member: %s  node: %s  -  force left the cluster!\n", member, server.Name)
			}
		}
	}
	if !validnode {
		fmt.Printf("On member: %s there is no Node with status code: %v\n", member, nodeStatusCode)
	}
}

func main() {
	flag.StringVar(&hostdiscovery, []string{"hd", "-host-discovery"}, "aws", "Host discovery. 'consul' or 'aws' or 'stdin'.")

	flag.StringVar(&url, []string{"u", "-url"}, "localhost", "Consul member endpoint. Default: localhost.")
	flag.StringVar(&port, []string{"p", "-port"}, "8500", "Consul members endpoint port. Default: 8500.")

	flag.StringVar(&awsregion, []string{"ar", "-aws-region"}, "eu-west-1", "AWS Region. Default: eu-west-1.")
	flag.Var(&tagsValues, []string{"tv", "-tag-value"}, "AWS tag and value. Usage '-tv tag:value'. It is repeatable.")

	flag.BoolVar(&listTargetHosts, []string{"lth", "-list-target-hosts"}, false, "List target hosts.")
	flag.BoolVar(&listNodeStatus, []string{"lns", "-list-node-status"}, false, "List nodes status.")
	flag.BoolVar(&listChecks, []string{"lchk", "-list-checks"}, false, "List checks.")
	flag.BoolVar(&listSrvInState, []string{"lsrvis", "-list-service-in-state"}, false, "List of services in specific state. Use it with --service-state.")
	flag.BoolVar(&listAllSrv, []string{"lasrv", "-list-all-services"}, false, "List all services.")

	flag.BoolVar(&deregisterSrv, []string{"drsrv", "-deregister-service"}, false, "Deregister service in specific state. Use it with --service-state.")
	flag.StringVar(&srvState, []string{"ss", "-service-state"}, "critical", "State of the service you wish to deregister. Default: critical.")

	flag.Var(&srvnameNode, []string{"dsn", "-deregister-srvname-node"}, "Deregister service node. Usage '-dsn serviceName:node'. It is repeatable.")
	flag.Var(&srvnameNodePortTag, []string{"rsnpt", "-register-srvname-node-port-tag"}, "Register service node. Usage '-rsnpt serviceName:node:port:tag'. It is repeatable.")

	flag.IntVar(&nsc, []string{"nsc", "-node-status-code"}, 4, "Consul node status code. Default: 4.")
	flag.BoolVar(&forceLeaveNode, []string{"fl", "-force-leave-node"}, false, "Force leave consul node. Use it with --node-status-code.")

	flag.BoolVar(&dryRun, []string{"d", "-dryrun"}, false, "Dryrun")
	flag.BoolVar(&ver, []string{"v", "-version"}, false, "Consul-cleaner Version.")
	flag.Parse()

	consulClient := connection(url, port)

	if ver == true {
		fmt.Printf("Consul-cleaner Version: %s\n", version)
		os.Exit(0)
	}

	switch hostdiscovery {
	case "consul":
		hosts = consulmembers(consulClient)
	case "aws":
		hosts = awshosts(awsregion, tagsValues)
	case "stdin":
		hosts = readHostsFromStdin()
	}

	if listTargetHosts {
		listTargetHost(hosts)
		os.Exit(0)
	}

	if rnptSet {
		registerService(connection(url, port), srvnameNodePortTag)
	}

	if dsnSet {
		deregisterNodes(connection(url, port), srvnameNode)
	}

	if deregisterSrv {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip, srvState string) {
				deregisterServiceInState(connection(ip, "8500"), srvState)
				wg.Done()
			}(ip, srvState)
		}
	}

	if listSrvInState {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip, srvState string) {
				listServicesInState(connection(ip, "8500"), srvState)
				wg.Done()
			}(ip, srvState)
		}
	}

	if listNodeStatus {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip string) {
				listNodeStat(connection(ip, "8500"))
				wg.Done()
			}(ip)
		}
	}

	if listChecks {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip string) {
				listCheck(connection(ip, "8500"))
				wg.Done()
			}(ip)
		}
	}

	if listAllSrv {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip string) {
				listServices(connection(ip, "8500"))
				wg.Done()
			}(ip)
		}
	}

	if forceLeaveNode {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip string) {
				forceLeaveBadNode(connection(ip, "8500"), nsc, ip)
				wg.Done()
			}(ip)
		}
	}

	wg.Wait()
}
