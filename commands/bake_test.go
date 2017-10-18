package commands_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	jhandacommands "github.com/pivotal-cf/jhanda/commands"
	"github.com/pivotal-cf/kiln/commands"
	"github.com/pivotal-cf/kiln/commands/fakes"
)

var _ = Describe("bake", func() {
	var (
		tileMaker *fakes.TileMaker
		bake      commands.Bake
	)

	BeforeEach(func() {
		tileMaker = &fakes.TileMaker{}
		bake = commands.NewBake(tileMaker)
	})

	Describe("Execute", func() {
		It("builds the tile", func() {
			err := bake.Execute([]string{
				"--stemcell-tarball", "some-stemcell-tarball",
				"--releases-directory", "some-release-tarball-directory",
				"--releases-directory", "other-release-tarball-directory",
				"--metadata", "some-metadata",
				"--version", "1.2.3",
				"--product-name", "cool-product-name",
				"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(tileMaker.MakeCallCount()).To(Equal(1))

			config := tileMaker.MakeArgsForCall(0)
			Expect(config).To(Equal(commands.BakeConfig{
				StemcellTarball:    "some-stemcell-tarball",
				ReleaseDirectories: []string{"some-release-tarball-directory", "other-release-tarball-directory"},
				Handcraft:          "some-metadata",
				Version:            "1.2.3",
				ProductName:        "cool-product-name",
				OutputFile:         "some-output-dir/cool-product-file-1.2.3-build.4",
			}))
		})

		Context("when migrations directory is provided", func() {
			It("builds the tile", func() {
				err := bake.Execute([]string{
					"--stemcell-tarball", "some-stemcell-tarball",
					"--releases-directory", "some-release-tarball-directory",
					"--migrations-directory", "some-migrations-directory",
					"--metadata", "some-metadata",
					"--version", "1.2.3",
					"--product-name", "cool-product-name",
					"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(tileMaker.MakeCallCount()).To(Equal(1))

				config := tileMaker.MakeArgsForCall(0)
				Expect(config).To(Equal(commands.BakeConfig{
					StemcellTarball:      "some-stemcell-tarball",
					ReleaseDirectories:   []string{"some-release-tarball-directory"},
					Handcraft:            "some-metadata",
					Version:              "1.2.3",
					ProductName:          "cool-product-name",
					MigrationDirectories: []string{"some-migrations-directory"},
					OutputFile:           "some-output-dir/cool-product-file-1.2.3-build.4",
				}))
			})

			Context("when the migration directory is specified multiple times", func() {
				It("builds the tile", func() {
					err := bake.Execute([]string{
						"--stemcell-tarball", "some-stemcell-tarball",
						"--releases-directory", "some-release-tarball-directory",
						"--migrations-directory", "some-migrations-directory",
						"--migrations-directory", "some-other-migrations-directory",
						"--metadata", "some-metadata",
						"--version", "1.2.3",
						"--product-name", "cool-product-name",
						"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
					})

					Expect(err).NotTo(HaveOccurred())
					Expect(tileMaker.MakeCallCount()).To(Equal(1))

					config := tileMaker.MakeArgsForCall(0)
					Expect(config).To(Equal(commands.BakeConfig{
						StemcellTarball:      "some-stemcell-tarball",
						ReleaseDirectories:   []string{"some-release-tarball-directory"},
						Handcraft:            "some-metadata",
						Version:              "1.2.3",
						ProductName:          "cool-product-name",
						MigrationDirectories: []string{"some-migrations-directory", "some-other-migrations-directory"},
						OutputFile:           "some-output-dir/cool-product-file-1.2.3-build.4",
					}))

				})

			})
		})

		It("builds the tile", func() {
			err := bake.Execute([]string{
				"--stemcell-tarball", "some-stemcell-tarball",
				"--releases-directory", "some-release-tarball-directory",
				"--metadata", "some-metadata",
				"--version", "1.2.3",
				"--product-name", "cool-product-name",
				"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(tileMaker.MakeCallCount()).To(Equal(1))

			config := tileMaker.MakeArgsForCall(0)
			Expect(config).To(Equal(commands.BakeConfig{
				StemcellTarball:    "some-stemcell-tarball",
				ReleaseDirectories: []string{"some-release-tarball-directory"},
				Handcraft:          "some-metadata",
				Version:            "1.2.3",
				ProductName:        "cool-product-name",
				OutputFile:         "some-output-dir/cool-product-file-1.2.3-build.4",
			}))
		})

		Context("failure cases", func() {
			Context("when the release-tarball flag is missing", func() {
				It("returns an error", func() {
					err := bake.Execute([]string{
						"--stemcell-tarball", "some-stemcell-tarball",
						"--metadata", "some-metadata",
						"--version", "1.2.3",
						"--product-name", "cool-product-name",
						"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
						"--stub-releases",
					})

					Expect(err).To(MatchError("Please specify release tarballs directory with the --releases-directory parameter"))
				})
			})

			Context("when the stemcell-tarball flag is missing", func() {
				It("returns an error", func() {
					err := bake.Execute([]string{
						"--releases-directory", "some-release-tarball-directory",
						"--metadata", "some-metadata",
						"--version", "1.2.3",
						"--product-name", "cool-product-name",
						"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
						"--stub-releases",
					})

					Expect(err).To(MatchError("--stemcell-tarball is a required parameter"))
				})
			})

			Context("when the metadata flag is missing", func() {
				It("returns an error", func() {
					err := bake.Execute([]string{
						"--releases-directory", "some-release-tarball-directory",
						"--stemcell-tarball", "some-stemcell-tarball",
						"--version", "1.2.3",
						"--product-name", "cool-product-name",
						"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
						"--stub-releases",
					})

					Expect(err).To(MatchError("--metadata is a required parameter"))
				})
			})

			Context("when the version flag is missing", func() {
				It("returns an error", func() {
					err := bake.Execute([]string{
						"--releases-directory", "some-release-tarball-directory",
						"--stemcell-tarball", "some-stemcell-tarball",
						"--metadata", "some-metadata",
						"--product-name", "cool-product-name",
						"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
						"--stub-releases",
					})

					Expect(err).To(MatchError("--version is a required parameter"))
				})
			})

			Context("when the product-name flag is missing", func() {
				It("returns an error", func() {
					err := bake.Execute([]string{
						"--releases-directory", "some-release-tarball-directory",
						"--stemcell-tarball", "some-stemcell-tarball",
						"--metadata", "some-metadata",
						"--version", "1.2.3",
						"--output-file", "some-output-dir/cool-product-file-1.2.3-build.4",
						"--stub-releases",
					})

					Expect(err).To(MatchError("--product-name is a required parameter"))
				})
			})

			Context("when the output-file flag is missing", func() {
				It("returns an error", func() {
					err := bake.Execute([]string{
						"--releases-directory", "some-release-tarball-directory",
						"--stemcell-tarball", "some-stemcell-tarball",
						"--metadata", "some-metadata",
						"--product-name", "cool-product-name",
						"--version", "1.2.3",
						"--stub-releases",
					})

					Expect(err).To(MatchError("--output-file is a required parameter"))
				})
			})

		})
	})

	Describe("Usage", func() {
		It("returns usage information for the command", func() {
			command := commands.NewBake(nil)
			Expect(command.Usage()).To(Equal(jhandacommands.Usage{
				Description:      "Builds a tile to be uploaded to OpsMan from provided inputs.",
				ShortDescription: "builds a tile",
				Flags:            command.Options,
			}))
		})
	})
})
