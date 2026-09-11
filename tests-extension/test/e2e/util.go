package router

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

func getRandomString() string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	buffer := make([]byte, 8)
	for index := range buffer {
		buffer[index] = chars[rand.Intn(len(chars))]
	}
	return string(buffer)
}

func waitForPodWithLabelReady(oc *exutil.CLI, ns, label string) error {
	return wait.Poll(5*time.Second, 3*time.Minute, func() (bool, error) {
		status, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", ns, "-l", label, `-ojsonpath={.items[*].status.conditions[?(@.type=="Ready")].status}`).Output()
		e2e.Logf("the Ready status of pod is %v", status)
		if err != nil || status == "" {
			e2e.Logf("failed to get pod status: %v, retrying...", err)
			return false, nil
		}
		if strings.Contains(status, "False") {
			e2e.Logf("the pod Ready status not met; wanted True but got %v, retrying...", status)
			return false, nil
		}
		return true, nil
	})
}

func ensurePodWithLabelReady(oc *exutil.CLI, ns, label string) {
	err := waitForPodWithLabelReady(oc, ns, label)
	if err != nil {
		output, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", ns, "-l", label).Output()
		e2e.Logf("All pods with label %v are:\n%v", label, output)
		logs, _ := oc.AsAdmin().WithoutNamespace().Run("logs").Args("-n", ns, "-l", label, "--tail=10").Output()
		e2e.Logf("The logs of all labeled pods are:\n%v", logs)
	}
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("max time reached but the pods with label %v are not ready", label))
}

func waitForResourceToDisappear(oc *exutil.CLI, ns, rsname string) error {
	return wait.Poll(20*time.Second, 5*time.Minute, func() (bool, error) {
		status, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(rsname, "-n", ns).Output()
		e2e.Logf("check resource %v and got: %v", rsname, status)
		primary := false
		if err != nil {
			if strings.Contains(status, "NotFound") {
				e2e.Logf("the resource is disappeared!")
				primary = true
			} else {
				e2e.Logf("failed to get the resource: %v, retrying...", err)
			}
		} else {
			e2e.Logf("the resource is still there, retrying...")
		}
		return primary, nil
	})
}

func patchGlobalResourceAsAdmin(oc *exutil.CLI, resource, patch string) {
	patchOut, err := oc.AsAdmin().WithoutNamespace().Run("patch").Args(resource, "--patch="+patch, "--type=json").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The output from the patch is:- %q ", patchOut)
}

func getByJsonPath(oc *exutil.CLI, ns, resource, jsonPath string) string {
	output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", ns, resource, "-o=jsonpath="+jsonPath).Output()
	if err != nil {
		e2e.Logf("the error is: %v", err.Error())
	}
	e2e.Logf("the output filtered by jsonpath is: %v", output)
	return output
}

func getByLabelAndJsonPath(oc *exutil.CLI, ns, resource, label, jsonPath string) string {
	output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", ns, resource, "-l", label, "-ojsonpath="+jsonPath).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("the output filtered by label and jsonpath is: %v", output)
	return output
}

func getNodeNameByPod(oc *exutil.CLI, namespace string, podName string) string {
	nodeName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", podName, "-n", namespace, "-o=jsonpath={.spec.nodeName}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The nodename for pod %s in namespace %s is %s", podName, namespace, nodeName)
	return nodeName
}

func getPodListByLabel(oc *exutil.CLI, namespace string, label string) []string {
	var podList []string
	podNameAll, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", namespace, "pod", "-l", label, "-ojsonpath={.items..metadata.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	podList = strings.Split(podNameAll, " ")
	e2e.Logf("The pod list is %v", podList)
	return podList
}

func getDNSPodName(oc *exutil.CLI) string {
	ns := "openshift-dns"
	podName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", ns, "pods", "-l", "dns.operator.openshift.io/daemonset-dns=default", "-o=jsonpath={.items[0].metadata.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("The DNS pod name is: %v", podName)
	return podName
}

func pollReadDnsCorefile(oc *exutil.CLI, dnsPodName, searchString1, grepOption, searchString2 string) string {
	e2e.Logf("Polling and search dns Corefile")
	ns := "openshift-dns"
	cmd1 := fmt.Sprintf("grep \"%s\" /etc/coredns/Corefile %s | grep \"%s\"", searchString1, grepOption, searchString2)
	cmd2 := fmt.Sprintf("grep \"%s\" /etc/coredns/Corefile %s", searchString1, grepOption)

	waitErr := wait.PollImmediate(5*time.Second, 120*time.Second, func() (bool, error) {
		hackAnnotatePod(oc, ns, dnsPodName)
		_, err := oc.AsAdmin().WithoutNamespace().Run("exec").Args("-n", ns, dnsPodName, "--", "bash", "-c", cmd1).Output()
		if err != nil {
			e2e.Logf("string not found, wait and try again...")
			return false, nil
		}
		return true, nil
	})
	if waitErr != nil {
		output, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-n", ns, "-l", "dns.operator.openshift.io/daemonset-dns=default").Output()
		e2e.Logf("All current dns pods are:\n%v", output)
		output, _ = oc.AsAdmin().WithoutNamespace().Run("exec").Args("-n", ns, dnsPodName, "--", "bash", "-c", "cat /etc/coredns/Corefile").Output()
		e2e.Logf("The existing Corefile is: %v", output)
	}
	compat_otp.AssertWaitPollNoErr(waitErr, fmt.Sprintf("reached max time allowed but Corefile is not updated"))
	output, err := oc.AsAdmin().WithoutNamespace().Run("exec").Args("-n", ns, dnsPodName, "--", "bash", "-c", cmd2).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("the part of Corefile that matching \"%s\" is: %v", searchString1, output)
	return output
}

func hackAnnotatePod(oc *exutil.CLI, ns, podName string) {
	hackAnnotation := "ne-testing-hack=" + getRandomString()
	oc.AsAdmin().WithoutNamespace().Run("annotate").Args("pod", podName, "-n", ns, hackAnnotation, "--overwrite").Execute()
}

func ensureClusterOperatorNormal(oc *exutil.CLI, coName string, healthyThreshold int, totalWaitTime time.Duration) {
	count := 0
	printCount := 0
	jsonPath := `{.status.conditions[?(@.type=="Available")].status}{.status.conditions[?(@.type=="Progressing")].status}{.status.conditions[?(@.type=="Degraded")].status}`

	e2e.Logf("waiting for CO %v back to normal status......", coName)
	waitErr := wait.PollImmediate(5*time.Second, totalWaitTime*time.Second, func() (bool, error) {
		status := getByJsonPath(oc, "default", "co/"+coName, jsonPath)
		primary := false
		printCount++
		if strings.Compare(status, "TrueFalseFalse") == 0 {
			count++
			if count == healthyThreshold {
				e2e.Logf("got %v successive good status (%v), the CO is stable!", count, status)
				primary = true
			} else {
				e2e.Logf("got %v successive good status (%v), try again...", count, status)
			}
		} else {
			count = 0
			if printCount%10 == 1 {
				e2e.Logf("CO status is still abnormal (%v), wait and try again...", status)
			}
		}
		return primary, nil
	})
	if waitErr != nil {
		output := getByJsonPath(oc, "default", "co/"+coName, "{.status.conditions}")
		e2e.Logf("The co %v is abnormal and here is status: %v", coName, output)
		if coName == "ingress" {
			output, _ = oc.AsAdmin().WithoutNamespace().Run("describe").Args("-n", "openshift-ingress", "service", "router-default").Output()
			e2e.Logf("The output of describe router-default service: %v", output)
		}
	}
	compat_otp.AssertWaitPollNoErr(waitErr, fmt.Sprintf("reached max time allowed but CO %v is still abnoraml.", coName))
}

func forceOnlyOneDnsPodExist(oc *exutil.CLI) string {
	ns := "openshift-dns"
	dnsPodLabel := "dns.operator.openshift.io/daemonset-dns=default"
	dnsNodeSelector := `[{"op":"replace", "path":"/spec/nodePlacement/nodeSelector", "value":{"nid-dns-testing":"true"}}]`
	oc.AsAdmin().WithoutNamespace().Run("label").Args("node", "-l", "nid-dns-testing=true", "nid-dns-testing-").Execute()
	podList := getAllDNSPodsNames(oc)
	if len(podList) == 1 {
		e2e.Logf("Found only one dns-default pod and it looks like SNO cluster. Continue the test...")
	} else {
		dnsPodName := getRandomElementFromList(podList)
		nodeName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", dnsPodName, "-o=jsonpath={.spec.nodeName}", "-n", ns).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Find random dns pod '%s' and its node '%s' which will be used for the following testing", dnsPodName, nodeName)
		_, err = oc.AsAdmin().WithoutNamespace().Run("label").Args("node", nodeName, "nid-dns-testing=true").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		patchGlobalResourceAsAdmin(oc, "dnses.operator.openshift.io/default", dnsNodeSelector)
		err1 := waitForResourceToDisappear(oc, ns, "pod/"+dnsPodName)
		if err1 != nil {
			output, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-n", ns, "-l", dnsPodLabel).Output()
			e2e.Logf("All current dns pods are:\n%v", output)
		}
		compat_otp.AssertWaitPollNoErr(err1, fmt.Sprintf("max time reached but pod %s is not terminated", dnsPodName))
		err2 := waitForPodWithLabelReady(oc, ns, dnsPodLabel)
		if err2 != nil {
			output, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-n", ns, "-l", dnsPodLabel).Output()
			e2e.Logf("All current dns pods are:\n%v", output)
		}
		compat_otp.AssertWaitPollNoErr(err2, fmt.Sprintf("max time reached but no dns pod ready"))
	}
	return getDNSPodName(oc)
}

func deleteDnsOperatorToRestore(oc *exutil.CLI) {
	_, err := oc.AsAdmin().WithoutNamespace().Run("delete").Args("dnses.operator.openshift.io/default").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	ensureClusterOperatorNormal(oc, "dns", 2, 120)
	oc.AsAdmin().WithoutNamespace().Run("label").Args("node", "-l", "nid-dns-testing=true", "nid-dns-testing-").Execute()
}

func getAllDNSPodsNames(oc *exutil.CLI) []string {
	ns := "openshift-dns"
	label := "dns.operator.openshift.io/daemonset-dns=default"
	dnsPods := getByLabelAndJsonPath(oc, ns, "pod", label, "{.items[*].metadata.name}")
	return strings.Split(dnsPods, " ")
}

func getRandomElementFromList(list []string) string {
	return list[rand.Intn(len(list))]
}

func waitForRangeOfPodsToDisappear(oc *exutil.CLI, resource string, podList []string) {
	for _, podName := range podList {
		err := waitForResourceToDisappear(oc, resource, "pod/"+podName)
		compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("%s pod %s is NOT deleted", resource, podName))
	}
}

func waitForOutputEquals(oc *exutil.CLI, ns, resourceName, jsonPath, expected string, args ...interface{}) {
	waitDuration := 180 * time.Second
	for _, arg := range args {
		duration, ok := arg.(time.Duration)
		if ok {
			waitDuration = duration
		}
	}

	waitErr := wait.PollImmediate(5*time.Second, waitDuration, func() (bool, error) {
		output := getByJsonPath(oc, ns, resourceName, jsonPath)
		if output == expected {
			return true, nil
		}
		e2e.Logf("The output of jsonpath does NOT equal the expected string: %v, retrying...", expected)
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(waitErr, fmt.Sprintf("max time reached but cannot find the expected string"))
}
