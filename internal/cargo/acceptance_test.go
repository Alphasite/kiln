package cargo_test

import (
	"io/ioutil"

	"github.com/pivotal-cf/kiln/internal/cargo"
	"github.com/pivotal-cf/kiln/internal/cargo/bosh"
	"github.com/pivotal-cf/kiln/internal/proofing"
	yaml "gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("Acceptance", func() {
	var generator cargo.Generator

	BeforeEach(func() {
		generator = cargo.NewGenerator()
	})

	XIt("can render a PAS manifest", func() {
		productTemplate, err := proofing.Parse("fixtures/acceptance/metadata.yml")
		Expect(err).NotTo(HaveOccurred())

		var stemcells []bosh.Stemcell
		var availabilityZones []string

		manifest := generator.Execute("cf-1234", productTemplate, stemcells, availabilityZones)

		actualManifest, err := yaml.Marshal(manifest)
		Expect(err).NotTo(HaveOccurred())

		expectedManifest, err := ioutil.ReadFile("fixtures/acceptance/manifest.yml")
		Expect(err).NotTo(HaveOccurred())
		Expect(actualManifest).To(HelpfullyMatchYAML(string(expectedManifest)))
	})
})