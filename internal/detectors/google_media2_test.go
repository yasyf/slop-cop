package detectors

import "testing"

func TestDetectGoogleUnqualifiedCodeElement(t *testing.T) {
	runHits(t, "unqualified-code-element", DetectGoogleUnqualifiedCodeElement, []hitCase{
		{"bare filename after determiner", "Edit the example.yaml and restart the daemon.", true},
		{"identifier opens sentence", "`Widget` accepts a timeout in milliseconds.", true},
		{"filename ends sentence after preposition", "Modify the default values in `build.sh`.", true},
		{"qualified filename", "Edit the `example.yaml` file and restart the daemon.", false},
		{"qualified identifier", "The `Widget` class accepts a timeout in milliseconds.", false},
		{"qualified filename after preposition", "Modify the default values in the `build.sh` file.", false},
		{"qualified flag", "Set the `--project` flag to the ID of your project.", false},
		{"plural category noun", "Edit the config.yaml files before you deploy.", false},
		{"reference entry", "- `--timeout`: Sets the deadline for each request.", false},
		{"list term", "- `internal/rules/base.go` holds the base layer.", false},
		{"attributive compound", "The watch prints a `--budget`-capped excerpt.", false},
		{"module path fragment", "The build pins go.yaml.in/yaml/v4 to the tagged release.", false},
		{"predicate after is", "The output file is `results.json`.", false},
		{"filename inside a link", "For details, see [the config.yaml reference](https://example.com/config.yaml).", false},
		{"filename inside a URL", "Download the archive from https://example.com/setup.sh and run it.", false},
		{"lowercase value span", "The `timeout` value defaults to 30 seconds.", false},
		{"plain prose", "Run the following command to restart the daemon and check the logs.", false},
		{"plain prose with numbers", "The service returns a 404 error when the request is missing a header.", false},
	})
}

func TestDetectGoogleUsEnglishSpelling(t *testing.T) {
	runHits(t, "us-english-spelling", DetectGoogleUsEnglishSpelling, []hitCase{
		{"customise colour initialise", "Customise the colour of the label before you initialise the client.", true},
		{"whilst centres", "Whilst the job runs, the service centres the workload across zones.", true},
		{"dialogue box", "The dialogue box appears after the upload finishes.", true},
		{"lowercase grey", "The disabled row uses a grey background.", true},
		{"us forms", "Customize the color of the label before you initialize the client.", false},
		{"us forms again", "While the job runs, the service centers the workload across zones.", false},
		{"grey as a surname", "Contact Jane Grey for access to the staging cluster.", false},
		{"defence in an organization name", "Published by the AeroSpace and Defence Industries Association of Europe.", false},
		{"camel case identifier", "The getColour helper stays as written in the generated bindings.", false},
		{"us inflections", "Fulfill the request and enroll the user before the license expires.", false},
		{"plain prose", "The analysis of the program output shows one canceled job and a gray label.", false},
	})
}
