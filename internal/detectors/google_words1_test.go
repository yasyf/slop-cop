package detectors

import "testing"

func TestDetectGoogleAbbreviationBeforeExpansion(t *testing.T) {
	runHits(t, "abbreviation-before-expansion", DetectGoogleAbbreviationBeforeExpansion, []hitCase{
		{"iot first", "The IoT (internet of things) service is regional.", true},
		{"ttl first", "Set the TTL (time to live) to 300.", true},
		{"api first", "The API (application programming interface) is documented.", true},
		{"iot expanded first", "The internet of things (IoT) service is regional.", false},
		{"ttl expanded first", "Set the time to live (TTL) to 300.", false},
		{"unrelated parenthetical", "The HTTP (see RFC 7231) status codes are stable.", false},
		{"ordinary reference prose", "The service returns a 404 when the key is missing.", false},
		{"ordinary procedure", "Set the cache expiry to 300 seconds.", false},
	})
}

func TestDetectGoogleApostrophePlural(t *testing.T) {
	runHits(t, "apostrophe-plural", DetectGoogleApostrophePlural, []hitCase{
		{"acronym list", "API's, SKE's, and IDE's are all supported.", true},
		{"decade", "Tools from the 1990's.", true},
		{"product possessive", "Kubernetes's scheduler picks a node.", true},
		{"acronym plural verb", "VM's are billed per second.", true},
		{"acronyms fixed", "APIs, SKEs, and IDEs are all supported.", false},
		{"decade fixed", "Tools from the 1990s.", false},
		{"product rephrased", "The Kubernetes scheduler picks a node.", false},
		{"genuine possessive", "The API's response body is JSON.", false},
		{"ordinary reference prose", "The server returns HTTP 200.", false},
		{"ordinary procedure", "Configure the VM before you deploy.", false},
	})
}

func TestDetectGoogleCompoundSubjectAgreement(t *testing.T) {
	runHits(t, "compound-subject-agreement", DetectGoogleCompoundSubjectAgreement, []hitCase{
		{"authentication and authorization", "User authentication and authorization is processed by the security module.", true},
		{"latency and throughput", "Latency and throughput has improved.", true},
		{"plural verb", "User authentication and authorization are processed by the security module.", false},
		{"plural have", "Latency and throughput have improved.", false},
		{"prepositional pair", "The relationship between latency and throughput is nonlinear.", false},
		{"idiom", "Read and write is disabled.", false},
		{"each scope", "Each region and zone is billed separately.", false},
		{"separate clauses", "The cache warms and the entry is valid.", false},
		{"ordinary reference prose", "The scheduler assigns each pod to a node.", false},
		{"ordinary procedure", "Latency improved after the cache warmed.", false},
	})
}

func TestDetectGoogleDocSelfReference(t *testing.T) {
	runHits(t, "doc-self-reference", DetectGoogleDocSelfReference, []hitCase{
		{"this article", "You can find more examples in this article.", true},
		{"this doc", "See the troubleshooting section of this doc.", true},
		{"this page", "The limits described on this page apply per project.", true},
		{"this topic", "This topic covers the retry budget.", true},
		{"this document", "You can find more examples in this document.", false},
		{"this document again", "See the troubleshooting section of this document.", false},
		{"named type tutorial", "You can find more examples in this tutorial.", false},
		{"named type guide", "See the troubleshooting section of this guide.", false},
		{"ordinary reference prose", "The document describes the retry policy.", false},
		{"ordinary procedure", "Read the reference for the full flag list.", false},
	})
}

func TestDetectGoogleDroppedArticle(t *testing.T) {
	runHits(t, "dropped-article", DetectGoogleDroppedArticle, []hitCase{
		{"create vm instance", "## Create VM instance", true},
		{"delete service account", "## Delete service account", true},
		{"list item", "- Configure firewall rule for the cluster", true},
		{"with article", "## Create a VM instance", false},
		{"with the", "## Delete the service account", false},
		{"plural head", "## Create VM instances", false},
		{"gerund head", "## Enable audit logging", false},
		{"proper noun head", "## Configure Cloud Storage", false},
		{"single object", "## Deploy application", false},
		{"ordinary reference prose", "The installer creates a VM instance.", false},
		{"ordinary procedure", "Run the migration, then verify the health endpoint.", false},
	})
}

func TestDetectGoogleExpansionNumberMismatch(t *testing.T) {
	runHits(t, "expansion-number-mismatch", DetectGoogleExpansionNumberMismatch, []hitCase{
		{"plural expansion singular abbrev", "virtual machines (VM)", true},
		{"singular expansion plural abbrev", "virtual machine (VMs)", true},
		{"slo mismatch", "Define the service level objectives (SLO) for each tier.", true},
		{"both plural", "virtual machines (VMs)", false},
		{"both singular", "virtual machine (VM)", false},
		{"slo agreed", "Define the service level objectives (SLOs) for each tier.", false},
		{"unrelated parenthetical", "Enable the flag (beta) for testing.", false},
		{"ordinary reference prose", "The exporter writes Parquet files.", false},
		{"ordinary procedure", "Set the retention policy to 30 days.", false},
	})
}

func TestDetectGoogleIndefiniteArticleAbbreviation(t *testing.T) {
	runHits(t, "indefinite-article-abbreviation", DetectGoogleIndefiniteArticleAbbreviation, []hitCase{
		{"mixed articles", "Run a SQL query against an SAP export via a HTTP request to an URL.", true},
		{"a ssd", "Attach a SSD to a VM.", true},
		{"a iam", "Configure a IAM policy for the bucket.", true},
		{"an sql", "Run an SQL query against the replica.", true},
		{"mixed articles fixed", "Run a SQL query against an SAP export via an HTTP request to a URL.", false},
		{"an ssd", "Attach an SSD to a VM.", false},
		{"a rest api", "Send a REST request to the endpoint.", false},
		{"ordinary reference prose", "Send a request to the endpoint.", false},
		{"ordinary procedure", "Attach an encrypted volume to the instance.", false},
	})
}

func TestDetectGoogleLatinAbbreviation(t *testing.T) {
	runHits(t, "latin-abbreviation", DetectGoogleLatinAbbreviation, []hitCase{
		{"eg", "Set a retention policy, e.g. 30 days.", true},
		{"ie", "The primary region, i.e. the one you selected first, receives writes.", true},
		{"for instance", "For instance the retry budget is per host.", true},
		{"comma form", "Set a retention policy, e.g., 30 days.", true},
		{"missing comma", "For example the retry budget is per host.", true},
		{"eg fixed", "Set a retention policy, for example, 30 days.", false},
		{"ie fixed", "The primary region, that is, the one you selected first, receives writes.", false},
		{"for example fixed", "For example, the retry budget is per host.", false},
		{"instance as noun", "See the reference for instance types you can attach.", false},
		{"mid-sentence for example", "See the reference for example configurations.", false},
		{"ordinary reference prose", "The client retries three times before failing.", false},
		{"ordinary procedure", "Set the retention policy to 30 days.", false},
	})
}

func TestDetectGoogleNeitherNorPairing(t *testing.T) {
	runHits(t, "neither-nor-pairing", DetectGoogleNeitherNorPairing, []hitCase{
		{"neither client or server", "Neither the client or the server retries the request.", true},
		{"neither latency or throughput", "The flag affects neither latency or throughput.", true},
		{"neither nor", "Neither the client nor the server retries the request.", false},
		{"neither nor inline", "The flag affects neither latency nor throughput.", false},
		{"either or", "Either the client or the server retries.", false},
		{"ordinary reference prose", "The flag affects latency or throughput.", false},
		{"ordinary procedure", "Set either the timeout or the retry budget.", false},
	})
}
