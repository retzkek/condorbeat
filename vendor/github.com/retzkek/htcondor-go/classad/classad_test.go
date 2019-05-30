package classad

import (
	"strings"
	"testing"
)

func TestReadClassAd_bad(t *testing.T) {
	for _, s := range badClassads {
		_, err := ReadClassAds(strings.NewReader(s))
		if err == nil {
			t.Errorf("expected error. ClassAd:\n%s", s)
		}
	}
}

func TestReadClassAd_good(t *testing.T) {
	ads, err := ReadClassAds(strings.NewReader(classads))
	if err != nil {
		t.Error(err)
	}
	if len(ads) != classadsLen {
		t.Errorf("expected %d classads, read %d", classadsLen, len(ads))
	}
	for _, ad := range ads {
		t.Log(ad.Strings())
	}
}

func TestStreamClassAds_bad(t *testing.T) {
	for _, s := range badClassads {
		ch := make(chan ClassAd)
		errors := make(chan error)
		go StreamClassAds(strings.NewReader(s), ch, errors)
		n := 0
		for {
			select {
			case ad, ok := <-ch:
				if ok {
					t.Log(ad.Strings())
				} else {
					ch = nil
				}
			case err, ok := <-errors:
				if ok {
					t.Log(err)
					n++
				} else {
					errors = nil
				}
			default:
			}
			if ch == nil && errors == nil {
				break
			}
		}
		if n == 0 {
			t.Errorf("expected error. ClassAd:\n%s", s)
		}
	}
}

func TestStreamClassAds_good(t *testing.T) {
	ch := make(chan ClassAd)
	errors := make(chan error)
	go StreamClassAds(strings.NewReader(classads), ch, errors)
	//if err != nil {
	//	t.Error(err)
	//}
	n := 0
	for {
		select {
		case ad, ok := <-ch:
			if ok {
				t.Log(ad.Strings())
				n++
			} else {
				ch = nil
			}
		case err, ok := <-errors:
			if ok {
				t.Error(err)
			} else {
				errors = nil
			}
		}
		// break once both channels are closed
		if ch == nil && errors == nil {
			break
		}
	}
	if n != classadsLen {
		t.Errorf("expected %d classads, read %d", classadsLen, n)
	}
}

var badClassads = []string{
	`foo
bar`,
	`foo = bar
baz`,
}

var classadsLen = 2
var classads = `
Requirements = ( ( ( Arch == "X86_64" ) || ( Arch == "INTEL" ) ) && ( target.IS_Glidein == true ) && ( DesiredOS =?= NULL || stringlistimember(Target.IFOS_installed,DesiredOS) ) ) && ( TARGET.OpSys == "LINUX" ) && ( TARGET.Disk >= RequestDisk ) && ( TARGET.Memory >= RequestMemory ) && ( TARGET.HasFileTransfer )
ClusterId = 14158503
WhenToTransferOutput = "ON_EXIT_OR_EVICT"
MachineAttrGLIDEIN_Site0 = "FNAL_GPGrid"
LastMatchTime = 1486408188
MATCH_GLIDEIN_Schedd = "cmsgwms-factory.fnal.gov"
JOB_EXPECTED_MAX_LIFETIME = 28800
CompletionDate = 1486408342
RecentBlockWrites = 0
MATCH_EXP_JOB_GLIDEIN_SiteWMS_Queue = "gpce01.fnal.gov"
x509UserProxyVOName = "dune"
MATCH_GLIDEIN_SiteWMS = "HTCondor"
BufferSize = 524288
JobsubClientVersion = "1.2.3"
JOB_GLIDEIN_ProcId = "$$(GLIDEIN_ProcId:Unknown)"
StartdPrincipal = "execute-side@matchsession/131.225.216.41"
MATCH_EXP_JOB_GLIDEIN_Site = "FNAL_GPGrid"
TargetType = "Machine"
MATCH_EXP_JOB_GLIDEIN_ProcId = "9"
LeaveJobInQueue = false
x509UserProxyExpiration = 1486494487
RecentStatsLifetimeStarter = 144
JobNotification = 3
Owner = "jmalbos"
CondorPlatform = "$CondorPlatform: X86_64-CentOS_6.7 $"
JOB_GLIDEIN_Entry_Name = "$$(GLIDEIN_Entry_Name:Unknown)"
CommittedTime = 154
x509userproxy = "/var/lib/jobsub/creds/proxies/dune/x509cc_jmalbos_Analysis"
QDate = 1486408089
JobLeaseDuration = 3600
RecentBlockWriteKbytes = 0
TransferIn = false
ExitStatus = 0
NumJobMatches = 1
RootDir = "/"
JobCurrentStartDate = 1486408188
GlobalJobId = "fifebatch1.fnal.gov#14158503.10#1486408089"
CurrentHosts = 0
RemoteSysCpu = 6.0
TotalSuspensions = 0
x509UserProxyFirstFQAN = "/dune/Role=Analysis/Capability=NULL"
WantCheckpoint = false
LastJobLeaseRenewal = 1486408342
JobsubClientDN = "/DC=org/DC=cilogon/C=US/O=Fermi National Accelerator Laboratory/OU=People/CN=Justo Martin-albo simon/CN=UID:jmalbos"
LastRemoteHost = "slot1@glidein_3324402_136032460@fnpc4108.fnal.gov"

Requirements = ( ( ( Arch == "X86_64" ) || ( Arch == "INTEL" ) ) && ( target.IS_Glidein == true ) && ( DesiredOS =?= NULL || stringlistimember(Target.IFOS_installed,DesiredOS) ) && ( stringListsIntersect(toUpper(target.HAS_usage_model),toUpper(my.DESIRED_usage_model)) ) ) && ( TARGET.OpSys == "LINUX" ) && ( TARGET.Disk >= RequestDisk ) && ( TARGET.Memory >= RequestMemory ) && ( TARGET.HasFileTransfer )
ClusterId = 14155293
WhenToTransferOutput = "ON_EXIT_OR_EVICT"
MATCH_GLIDEIN_Schedd = "cmsgwms-factory.fnal.gov"
LastMatchTime = 1486401427
MachineAttrGLIDEIN_Site0 = "FNAL_GPGrid"
JOB_EXPECTED_MAX_LIFETIME = 85200
CompletionDate = 1486423594
RecentBlockWrites = 0
MATCH_EXP_JOB_GLIDEIN_SiteWMS_Queue = "gpce01.fnal.gov"
x509UserProxyVOName = "dune"
MATCH_GLIDEIN_SiteWMS = "HTCondor"
BufferSize = 524288
JobsubClientVersion = "1.2.3"
JOB_GLIDEIN_ProcId = "$$(GLIDEIN_ProcId:Unknown)"
StartdPrincipal = "execute-side@matchsession/131.225.209.142"
MATCH_EXP_JOB_GLIDEIN_Site = "FNAL_GPGrid"
TargetType = "Machine"
MATCH_EXP_JOB_GLIDEIN_ProcId = "0"
LeaveJobInQueue = false
x509UserProxyExpiration = 1486508061
RecentStatsLifetimeStarter = 1200
JobNotification = 3
Owner = "lebrun"
CondorPlatform = "$CondorPlatform: X86_64-CentOS_6.7 $"
JOB_GLIDEIN_Entry_Name = "$$(GLIDEIN_Entry_Name:Unknown)"
CommittedTime = 22167
x509userproxy = "/var/lib/jobsub/creds/proxies/dune/x509cc_lebrun_Analysis"
QDate = 1486401370
JobLeaseDuration = 3600
RecentBlockWriteKbytes = 0
TransferIn = false
ExitStatus = 0
NumJobMatches = 1
RootDir = "/"
JobCurrentStartDate = 1486401427
GlobalJobId = "fifebatch1.fnal.gov#14155293.202#1486401371"
CurrentHosts = 0
RemoteSysCpu = 9.0
TotalSuspensions = 0
x509UserProxyFirstFQAN = "/dune/Role=Analysis/Capability=NULL"
WantCheckpoint = false
LastJobLeaseRenewal = 1486423593
JobsubClientDN = "/DC=org/DC=cilogon/C=US/O=Fermi National Accelerator Laboratory/OU=People/CN=Paul Lebrun/CN=UID:lebrun"
LastRemoteHost = "slot1@glidein_3386316_395885625@fnpc9051.fnal.gov"
`
