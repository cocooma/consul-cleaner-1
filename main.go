package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/cocooma/awsdiscovery"
	flag "github.com/docker/docker/pkg/mflag"
	"github.com/hashicorp/consul/api"
)

var (
	str, url, port, srvState, awsregion, tag, tagvalue, hostdiscovery, line, version                                                                  string
	listTargetHosts, showMmemberStatus, listChecks, listAllSrv, deregisterSrv, listSrvInState, listNodeStatus, forceLeaveNode, dryRun, validnode, ver bool
	nsc                                                                                                                                               int
	hosts                                                                                                                                             []string
	wg                                                                                                                                                sync.WaitGroup
)

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

func awshosts(awsregion, tag, tagvalue string) []string {
	session := awsdiscovery.AwsSessIon(awsregion)
	filter := awsdiscovery.AwsFilter(tag, tagvalue)
	ips := awsdiscovery.AwsInstancePrivateIP(session, filter)
	return ips
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

func listServices(consulClient *api.Client) {
	services, _ := consulClient.Agent().Services()
	for _, service := range services {
		fmt.Println(service.ID, service.Service, service.Tags)
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

func deregisterService(consulConnection *api.Client, serviceCheckStatus string) {
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
	flag.StringVar(&hostdiscovery, []string{"hd", "-host-discovery"}, "aws", "Host discovery. 'consul' or 'aws' or 'stdin'")
	flag.StringVar(&url, []string{"u", "-url"}, "localhost", "Consul member endpoint. Default: localhost")
	flag.StringVar(&port, []string{"p", "-port"}, "8500", "Consul members endpoint port. Default: 8500")
	flag.StringVar(&awsregion, []string{"ar", "-aws-region"}, "eu-west-1", "AWS Region. Default: eu-west-1")
	flag.StringVar(&tag, []string{"t", "-tag"}, "", "AWS tag")
	flag.StringVar(&tagvalue, []string{"tv", "-tag-value"}, "", "AWS tag value")
	flag.IntVar(&nsc, []string{"nsc", "-node-status-code"}, 4, "Consul node status code. Default: 4")
	flag.BoolVar(&listTargetHosts, []string{"lth", "-list-target-hosts"}, false, "List target hosts")
	flag.BoolVar(&listNodeStatus, []string{"lns", "-list-node-status"}, false, "List nodes status")
	flag.BoolVar(&listChecks, []string{"lchk", "-list-checks"}, false, "List checks")
	flag.BoolVar(&listSrvInState, []string{"lsrvis", "-list-service-in-state"}, false, "List of services in specific state. Use it with --serviceState")
	flag.BoolVar(&listAllSrv, []string{"lasrv", "-list-all-services"}, false, "List of all services")
	flag.StringVar(&srvState, []string{"ss", "-service-state"}, "critical", "State of the service you wish to deregister. Default: critical")
	flag.BoolVar(&deregisterSrv, []string{"drsrv", "-deregister-service"}, false, "Deregister service. Use it with --serviceState")
	flag.BoolVar(&forceLeaveNode, []string{"fl", "-force-leave-node"}, false, "Force leave consul node. Use it with --nodeStatusCode")
	flag.BoolVar(&dryRun, []string{"d", "-dryrun"}, false, "Dryrun")
	flag.BoolVar(&ver, []string{"v", "-version"}, false, "Consul-cleaner Version")
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
		hosts = awshosts(awsregion, tag, tagvalue)
	case "stdin":
		hosts = readHostsFromStdin()
	}

	if listTargetHosts == true {
		listTargetHost(hosts)
		os.Exit(0)
	}

	if deregisterSrv == true {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip, srvState string) {
				deregisterService(connection(ip, "8500"), srvState)
				wg.Done()
			}(ip, srvState)
		}
	}

	if listSrvInState == true {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip, srvState string) {
				listServicesInState(connection(ip, "8500"), srvState)
				wg.Done()
			}(ip, srvState)
		}
	}

	if listNodeStatus == true {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip string) {
				listNodeStat(connection(ip, "8500"))
				wg.Done()
			}(ip)
		}
	}

	if listChecks == true {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip string) {
				listCheck(connection(ip, "8500"))
				wg.Done()
			}(ip)
		}
	}

	if listAllSrv == true {
		wg.Add(len(hosts))
		for _, ip := range hosts {
			go func(ip string) {
				listServices(connection(ip, "8500"))
				wg.Done()
			}(ip)
		}
	}

	if forceLeaveNode == true {
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
