package commands_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pivotal-cf/kiln/release"

	"gopkg.in/yaml.v2"

	"github.com/pivotal-cf/jhanda"
	"github.com/pivotal-cf/kiln/fetcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf/kiln/commands"
	"github.com/pivotal-cf/kiln/commands/fakes"
	fetcherFakes "github.com/pivotal-cf/kiln/fetcher/fakes"
)

var _ = Describe("Fetch", func() {
	var (
		fetch                       Fetch
		logger                      *log.Logger
		tmpDir                      string
		someKilnfilePath            string
		someKilnfileLockPath        string
		lockContents                string
		someReleasesDirectory       string
		fakeS3CompiledReleaseSource *fetcherFakes.ReleaseSource
		fakeBoshIOReleaseSource     *fetcherFakes.ReleaseSource
		fakeS3BuiltReleaseSource    *fetcherFakes.ReleaseSource
		fakeReleaseSources          []fetcher.ReleaseSource
		fakeLocalReleaseDirectory   *fakes.LocalReleaseDirectory
		releaseSourcesFactory       *fakes.ReleaseSourcesFactory

		fetchExecuteArgs []string
		fetchExecuteErr  error
	)

	const (
		s3CompiledReleaseSourceID = "s3-compiled"
		s3BuiltReleaseSourceID    = "s3-built"
		boshIOReleaseSourceID     = fetcher.BOSHIOReleaseSourceID
	)

	Describe("Execute", func() {
		BeforeEach(func() {
			logger = log.New(GinkgoWriter, "", 0)

			var err error
			tmpDir, err = ioutil.TempDir("", "fetch-test")

			someReleasesDirectory, err = ioutil.TempDir(tmpDir, "")
			Expect(err).NotTo(HaveOccurred())

			someKilnfilePath = filepath.Join(tmpDir, "Kilnfile")
			err = ioutil.WriteFile(someKilnfilePath, []byte(""), 0644)
			Expect(err).NotTo(HaveOccurred())

			someKilnfileLockPath = filepath.Join(tmpDir, "Kilnfile.lock")
			lockContents = `
---
releases:
- name: some-release
  version: "1.2.3"
  remote_source: ` + s3CompiledReleaseSourceID + `
  remote_path: my-remote-path
stemcell_criteria:
  os: some-os
  version: "4.5.6"
`

			fakeLocalReleaseDirectory = new(fakes.LocalReleaseDirectory)

			fakeS3CompiledReleaseSource = new(fetcherFakes.ReleaseSource)
			fakeS3CompiledReleaseSource.IDReturns(s3CompiledReleaseSourceID)
			fakeBoshIOReleaseSource = new(fetcherFakes.ReleaseSource)
			fakeBoshIOReleaseSource.IDReturns(boshIOReleaseSourceID)
			fakeS3BuiltReleaseSource = new(fetcherFakes.ReleaseSource)
			fakeS3BuiltReleaseSource.IDReturns(s3BuiltReleaseSourceID)

			fetchExecuteArgs = []string{
				"--releases-directory", someReleasesDirectory,
				"--kilnfile", someKilnfilePath,
			}
			releaseSourcesFactory = new(fakes.ReleaseSourcesFactory)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		JustBeforeEach(func() {
			fakeReleaseSources = []fetcher.ReleaseSource{fakeS3CompiledReleaseSource, fakeBoshIOReleaseSource, fakeS3BuiltReleaseSource}
			releaseSourcesFactory.ReleaseSourcesReturns(fakeReleaseSources)

			err := ioutil.WriteFile(someKilnfileLockPath, []byte(lockContents), 0644)
			Expect(err).NotTo(HaveOccurred())
			fetch = NewFetch(logger, releaseSourcesFactory, fakeLocalReleaseDirectory)

			fetchExecuteErr = fetch.Execute(fetchExecuteArgs)
		})

		When("a local compiled release exists", func() {
			const (
				expectedStemcellOS      = "fooOS"
				expectedStemcellVersion = "0.2.0"
			)
			var (
				releaseID                               release.ReleaseID
				releaseOnDisk                           release.SatisfyingLocalRelease
				actualStemcellOS, actualStemcellVersion string
			)
			BeforeEach(func() {
				releaseID = release.ReleaseID{Name: "some-release", Version: "0.1.0"}
				fakeS3CompiledReleaseSource.DownloadReleasesReturns(
					[]release.LocalRelease{
						{
							ReleaseID: releaseID,
							LocalPath: fmt.Sprintf("releases/%s-%s-%s-%s.tgz", releaseID.Name, releaseID.Version, expectedStemcellOS, expectedStemcellVersion),
						},
					}, nil)
				lockContents = `---
releases:
- name: ` + releaseID.Name + `
  version: "` + releaseID.Version + `"
  remote_source: ` + s3CompiledReleaseSourceID + `
  remote_path: not-used
stemcell_criteria:
  os: ` + expectedStemcellOS + `
  version: "` + expectedStemcellVersion + `"`
				fetchExecuteArgs = append(fetchExecuteArgs, "--no-confirm")
			})

			When("the release was compiled with a different os", func() {
				BeforeEach(func() {
					releaseOnDisk = release.NewLocalCompiledRelease(releaseID, "different-os", expectedStemcellVersion,
						fmt.Sprintf("releases/%s-%s-%s-%s.tgz", releaseID.Name, releaseID.Version, actualStemcellOS, actualStemcellVersion))
					fakeLocalReleaseDirectory.GetLocalReleasesReturns(
						[]release.SatisfyingLocalRelease{releaseOnDisk},
						nil)
				})

				It("deletes the file from disk", func() {
					Expect(fetchExecuteErr).NotTo(HaveOccurred())

					Expect(fakeS3CompiledReleaseSource.DownloadReleasesCallCount()).To(Equal(1))

					Expect(fakeLocalReleaseDirectory.DeleteExtraReleasesCallCount()).To(Equal(1))
					extras, noConfirm := fakeLocalReleaseDirectory.DeleteExtraReleasesArgsForCall(0)
					Expect(noConfirm).To(Equal(true))
					Expect(extras).To(HaveLen(1))
					Expect(extras).To(ConsistOf(
						release.LocalRelease{ReleaseID: releaseOnDisk.ReleaseID, LocalPath: releaseOnDisk.LocalPath},
					))
				})
			})

			When("the release was compiled with a different version of the same os", func() {
				BeforeEach(func() {
					releaseOnDisk = release.NewLocalCompiledRelease(
						releaseID,
						expectedStemcellOS,
						"404",
						fmt.Sprintf("releases/%s-%s-%s-%s.tgz", releaseID.Name, releaseID.Version, actualStemcellOS, actualStemcellVersion),
					)

					fakeLocalReleaseDirectory.GetLocalReleasesReturns(
						[]release.SatisfyingLocalRelease{releaseOnDisk},
						nil)
				})

				It("deletes the file from disk", func() {
					Expect(fetchExecuteErr).NotTo(HaveOccurred())

					Expect(fakeS3CompiledReleaseSource.DownloadReleasesCallCount()).To(Equal(1))

					Expect(fakeLocalReleaseDirectory.DeleteExtraReleasesCallCount()).To(Equal(1))
					extras, noConfirm := fakeLocalReleaseDirectory.DeleteExtraReleasesArgsForCall(0)
					Expect(noConfirm).To(Equal(true))
					Expect(extras).To(HaveLen(1))
					Expect(extras).To(ConsistOf(
						release.LocalRelease{ReleaseID: releaseOnDisk.ReleaseID, LocalPath: releaseOnDisk.LocalPath},
					))
				})
			})
		})

		Context("starting with no releases but all can be downloaded from their source (happy path)", func() {
			var (
				s3CompiledReleaseID = release.ReleaseID{Name: "lts-compiled-release", Version: "1.2.4"}
				s3BuiltReleaseID    = release.ReleaseID{Name: "lts-built-release", Version: "1.3.9"}
				boshIOReleaseID     = release.ReleaseID{Name: "boshio-release", Version: "1.4.16"}
			)
			BeforeEach(func() {
				lockContents = `---
releases:
- name: lts-compiled-release
  version: "1.2.4"
  remote_source: ` + s3CompiledReleaseSourceID + `
  remote_path: some-s3-key
- name: lts-built-release
  version: "1.3.9"
  remote_source: ` + s3BuiltReleaseSourceID + `
  remote_path: some-other-s3-key
- name: boshio-release
  version: "1.4.16"
  remote_source: ` + boshIOReleaseSourceID + `
  remote_path: some-bosh-io-url
stemcell_criteria:
  os: some-os
  version: "30.1"
`
				fakeS3CompiledReleaseSource.DownloadReleasesReturns(
					[]release.LocalRelease{{ReleaseID: s3CompiledReleaseID, LocalPath: "local-path"}},
					nil)

				fakeS3BuiltReleaseSource.DownloadReleasesReturns(
					[]release.LocalRelease{{ReleaseID: s3BuiltReleaseID, LocalPath: "local-path2"}},
					nil)

				fakeBoshIOReleaseSource.DownloadReleasesReturns(
					[]release.LocalRelease{{ReleaseID: boshIOReleaseID, LocalPath: "local-path3"}},
					nil)

				fakeLocalReleaseDirectory.GetLocalReleasesReturns([]release.SatisfyingLocalRelease{}, nil)
			})

			It("completes successfully", func() {
				Expect(fetchExecuteErr).NotTo(HaveOccurred())
			})

			It("fetches compiled release from s3 compiled release source", func() {
				Expect(fakeS3CompiledReleaseSource.DownloadReleasesCallCount()).To(Equal(1))

				releasesDir, objects, threads := fakeS3CompiledReleaseSource.DownloadReleasesArgsForCall(0)
				Expect(releasesDir).To(Equal(someReleasesDirectory))
				Expect(threads).To(Equal(0))
				Expect(objects).To(ConsistOf(
					release.RemoteRelease{ReleaseID: s3CompiledReleaseID, RemotePath: "some-s3-key"},
				))
			})

			It("fetches built release from s3 built release source", func() {
				Expect(fakeS3BuiltReleaseSource.DownloadReleasesCallCount()).To(Equal(1))
				releasesDir, objects, threads := fakeS3BuiltReleaseSource.DownloadReleasesArgsForCall(0)
				Expect(releasesDir).To(Equal(someReleasesDirectory))
				Expect(threads).To(Equal(0))
				Expect(objects).To(ConsistOf(
					release.RemoteRelease{ReleaseID: s3BuiltReleaseID, RemotePath: "some-other-s3-key"},
				))
			})

			It("fetches bosh.io release from bosh.io release source", func() {
				Expect(fakeBoshIOReleaseSource.DownloadReleasesCallCount()).To(Equal(1))
				releasesDir, objects, threads := fakeBoshIOReleaseSource.DownloadReleasesArgsForCall(0)
				Expect(releasesDir).To(Equal(someReleasesDirectory))
				Expect(threads).To(Equal(0))
				Expect(objects).To(ConsistOf(
					release.RemoteRelease{ReleaseID: boshIOReleaseID, RemotePath: "some-bosh-io-url"},
				))
			})
		})

		Context("when all releases are already present in releases directory", func() {
			BeforeEach(func() {
				lockContents = `---
releases:
- name: some-release-from-local-dir
  version: "1.2.3"
  remote_source: ` + s3CompiledReleaseSourceID + `
  remote_path: not-used
stemcell_criteria:
  os: some-os
  version: "4.5.6"
`

				someLocalReleaseID := release.ReleaseID{
					Name:    "some-release-from-local-dir",
					Version: "1.2.3",
				}
				fakeLocalReleaseDirectory.GetLocalReleasesReturns([]release.SatisfyingLocalRelease{
					release.NewLocalCompiledRelease(someLocalReleaseID, "some-os", "4.5.6", "/path/to/some/release"),
				}, nil)
			})

			It("no-ops", func() {
				Expect(fetchExecuteErr).NotTo(HaveOccurred())

				Expect(fakeS3CompiledReleaseSource.DownloadReleasesCallCount()).To(Equal(0))
				Expect(fakeS3BuiltReleaseSource.DownloadReleasesCallCount()).To(Equal(0))
				Expect(fakeBoshIOReleaseSource.DownloadReleasesCallCount()).To(Equal(0))
			})
		})

		Context("when some releases are already present in output directory", func() {
			var (
				missingReleaseS3CompiledID   release.ReleaseID
				missingReleaseS3CompiledPath = "s3-key-some-missing-release-on-s3-compiled"
				missingReleaseBoshIOID       release.ReleaseID
				missingReleaseBoshIOPath     = "some-other-bosh-io-key"
				missingReleaseS3BuiltID      release.ReleaseID
				missingReleaseS3BuiltPath    = "s3-key-some-missing-release-on-s3-built"

				missingReleaseS3Compiled,
				missingReleaseBoshIO,
				missingReleaseS3Built release.RemoteRelease
			)
			BeforeEach(func() {
				lockContents = `---
releases:
- name: some-release
  version: "1.2.3"
  remote_source: ` + s3BuiltReleaseSourceID + `
  remote_path: not-used
- name: some-tiny-release
  version: "1.2.3"
  remote_source: ` + boshIOReleaseSourceID + `
  remote_path: not-used2
- name: some-missing-release-on-s3-compiled
  version: "4.5.6"
  remote_source: ` + s3CompiledReleaseSourceID + `
  remote_path: ` + missingReleaseS3CompiledPath + `
- name: some-missing-release-on-boshio
  version: "5.6.7"
  remote_source: ` + boshIOReleaseSourceID + `
  remote_path: ` + missingReleaseBoshIOPath + `
- name: some-missing-release-on-s3-built
  version: "8.9.0"
  remote_source: ` + s3BuiltReleaseSourceID + `
  remote_path: ` + missingReleaseS3BuiltPath + `
stemcell_criteria:
  os: some-os
  version: "4.5.6"`

				missingReleaseS3CompiledID = release.ReleaseID{Name: "some-missing-release-on-s3-compiled", Version: "4.5.6"}
				missingReleaseBoshIOID = release.ReleaseID{Name: "some-missing-release-on-boshio", Version: "5.6.7"}
				missingReleaseS3BuiltID = release.ReleaseID{Name: "some-missing-release-on-s3-built", Version: "8.9.0"}

				fakeLocalReleaseDirectory.GetLocalReleasesReturns([]release.SatisfyingLocalRelease{
					release.NewLocalCompiledRelease(
						release.ReleaseID{Name: "some-release", Version: "1.2.3"},
						"some-os",
						"4.5.6",
						"path/to/some/release",
					),
					// a release that has no compiled packages, such as consul-drain, will also have no stemcell criteria in release.MF.
					// we must make sure that we can match this kind of release properly to avoid unnecessary downloads.
					release.NewLocalBuiltRelease(
						release.ReleaseID{Name: "some-tiny-release", Version: "1.2.3"},
						"path/to/some/tiny/release",
					),
				}, nil)

				fakeS3CompiledReleaseSource.DownloadReleasesReturns([]release.LocalRelease{
					{ReleaseID: missingReleaseS3CompiledID, LocalPath: "local-path-1"},
				}, nil)

				fakeBoshIOReleaseSource.DownloadReleasesReturns([]release.LocalRelease{
					{ReleaseID: missingReleaseBoshIOID, LocalPath: "local-path-2"},
				}, nil)

				fakeS3BuiltReleaseSource.DownloadReleasesReturns([]release.LocalRelease{
					{ReleaseID: missingReleaseS3BuiltID, LocalPath: "local-path-3"},
				}, nil)

				missingReleaseS3Compiled = release.RemoteRelease{ReleaseID: missingReleaseS3CompiledID, RemotePath: missingReleaseS3CompiledPath}
				missingReleaseBoshIO = release.RemoteRelease{ReleaseID: missingReleaseBoshIOID, RemotePath: missingReleaseBoshIOPath}
				missingReleaseS3Built = release.RemoteRelease{ReleaseID: missingReleaseS3BuiltID, RemotePath: missingReleaseS3BuiltPath}
			})

			It("downloads only the missing releases", func() {
				Expect(fetchExecuteErr).NotTo(HaveOccurred())

				Expect(fakeS3CompiledReleaseSource.DownloadReleasesCallCount()).To(Equal(1))
				_, objects, _ := fakeS3CompiledReleaseSource.DownloadReleasesArgsForCall(0)
				Expect(objects).To(HaveLen(1))
				Expect(objects).To(ConsistOf(missingReleaseS3Compiled))

				Expect(fakeBoshIOReleaseSource.DownloadReleasesCallCount()).To(Equal(1))
				_, objects, _ = fakeBoshIOReleaseSource.DownloadReleasesArgsForCall(0)
				Expect(objects).To(HaveLen(1))
				Expect(objects).To(ConsistOf(missingReleaseBoshIO))

				Expect(fakeS3BuiltReleaseSource.DownloadReleasesCallCount()).To(Equal(1))
				_, objects, _ = fakeS3BuiltReleaseSource.DownloadReleasesArgsForCall(0)
				Expect(objects).To(HaveLen(1))
				Expect(objects).To(ConsistOf(missingReleaseS3Built))
			})

			Context("when download fails", func() {
				var (
					wrappedErr error
				)

				BeforeEach(func() {
					wrappedErr = errors.New("kaboom")
					fakeS3CompiledReleaseSource.DownloadReleasesReturns(
						nil,
						wrappedErr,
					)
				})

				It("returns an error", func() {
					Expect(fetchExecuteErr).To(HaveOccurred())
					Expect(fetchExecuteErr).To(MatchError(ContainSubstring("download failed")))
					Expect(errors.Is(fetchExecuteErr, wrappedErr)).To(BeTrue())
				})
			})
		})

		Context("when there are extra releases locally that are not in the Kilnfile.lock", func() {
			var (
				boshIOReleaseID = release.ReleaseID{Name: "some-release", Version: "1.2.3"}
				localReleaseID  = release.ReleaseID{Name: "some-extra-release", Version: "1.2.3"}
			)
			BeforeEach(func() {

				lockContents = `---
releases:
- name: some-release
  version: "1.2.3"
  remote_source: ` + s3CompiledReleaseSourceID + `
  remote_path: not-used
stemcell_criteria:
  os: some-os
  version: "4.5.6"
`
				fakeLocalReleaseDirectory.GetLocalReleasesReturns([]release.SatisfyingLocalRelease{
					release.NewLocalCompiledRelease(localReleaseID, "some-os", "4.5.6", "path/to/some/extra/release"),
				}, nil)

				fakeBoshIOReleaseSource.DownloadReleasesReturns(
					[]release.LocalRelease{{ReleaseID: boshIOReleaseID, LocalPath: "local-path"}},
					nil)

			})

			Context("in non-interactive mode", func() {
				BeforeEach(func() {
					fetchExecuteArgs = []string{
						"--releases-directory", someReleasesDirectory,
						"--kilnfile", someKilnfilePath,
						"--no-confirm",
					}
				})

				It("deletes the extra releases", func() {
					Expect(fetchExecuteErr).NotTo(HaveOccurred())

					Expect(fakeS3CompiledReleaseSource.DownloadReleasesCallCount()).To(Equal(1))

					Expect(fakeLocalReleaseDirectory.DeleteExtraReleasesCallCount()).To(Equal(1))

					extras, noConfirm := fakeLocalReleaseDirectory.DeleteExtraReleasesArgsForCall(0)
					Expect(extras).To(HaveLen(1))
					Expect(noConfirm).To(Equal(true))
					Expect(extras).To(ConsistOf(
						release.LocalRelease{
							ReleaseID: release.ReleaseID{Name: "some-extra-release", Version: "1.2.3"},
							LocalPath: "path/to/some/extra/release",
						},
					))
				})
			})

			Context("when multiple variable files are provided", func() {
				const TemplatizedKilnfileYMLContents = `
---
release_sources:
  - type: s3
    compiled: true
    bucket: $( variable "bucket" )
    region: $( variable "region" )
    access_key_id: $( variable "access_key" )
    secret_access_key: $( variable "secret_key" )
    regex: $( variable "regex" )
`

				var (
					someVariableFile, otherVariableFile *os.File
				)

				BeforeEach(func() {
					var err error

					someKilnfilePath = filepath.Join(tmpDir, "Kilnfile")
					err = ioutil.WriteFile(someKilnfilePath, []byte(TemplatizedKilnfileYMLContents), 0644)
					Expect(err).NotTo(HaveOccurred())

					someVariableFile, err = ioutil.TempFile(tmpDir, "variables-file1")
					Expect(err).NotTo(HaveOccurred())
					defer someVariableFile.Close()

					variables := map[string]string{
						"bucket": "my-releases",
					}
					data, err := yaml.Marshal(&variables)
					Expect(err).NotTo(HaveOccurred())
					n, err := someVariableFile.Write(data)
					Expect(err).NotTo(HaveOccurred())
					Expect(data).To(HaveLen(n))

					otherVariableFile, err = ioutil.TempFile(tmpDir, "variables-file2")
					Expect(err).NotTo(HaveOccurred())
					defer otherVariableFile.Close()

					variables = map[string]string{
						"access_key": "newkey",
						"secret_key": "newsecret",
						"regex":      `^2.5/.+/(?P<release_name>[a-z-_]+)-(?P<release_version>[0-9\.]+)-(?P<stemcell_os>[a-z-_]+)-(?P<stemcell_version>[\d\.]+)\.tgz$`,
					}
					data, err = yaml.Marshal(&variables)
					Expect(err).NotTo(HaveOccurred())

					n, err = otherVariableFile.Write(data)
					Expect(err).NotTo(HaveOccurred())
					Expect(data).To(HaveLen(n))

					fetchExecuteArgs = []string{
						"--releases-directory", someReleasesDirectory,
						"--kilnfile", someKilnfilePath,
						"--variables-file", someVariableFile.Name(),
						"--variables-file", otherVariableFile.Name(),
						"--variable", "region=north-east-1",
					}
				})

				It("interpolates variables from both files", func() {
					Expect(fetchExecuteErr).NotTo(HaveOccurred())
				})
			})

			Context("when # of download threads is specified", func() {
				BeforeEach(func() {
					fetchExecuteArgs = []string{
						"--releases-directory", someReleasesDirectory,
						"--kilnfile", someKilnfilePath,
						"--download-threads", "10",
					}
				})

				It("passes concurrency parameter to DownloadReleases", func() {
					Expect(fetchExecuteErr).NotTo(HaveOccurred())
					_, _, threads := fakeS3CompiledReleaseSource.DownloadReleasesArgsForCall(0)
					Expect(threads).To(Equal(10))
				})
			})

			Context("failure cases", func() {
				Context("kilnfile is missing", func() {
					It("returns an error", func() {
						badKilnfilePath := filepath.Join(tmpDir, "non-existent-Kilnfile")
						err := fetch.Execute([]string{
							"--releases-directory", someReleasesDirectory,
							"--kilnfile", badKilnfilePath,
						})
						Expect(err).To(MatchError(fmt.Sprintf("open %s: no such file or directory", badKilnfilePath)))
					})
				})
				Context("# of download threads is not a number", func() {
					It("returns an error", func() {
						err := fetch.Execute([]string{
							"--releases-directory", someReleasesDirectory,
							"--kilnfile", someKilnfilePath,
							"--download-threads", "not-a-number",
						})
						Expect(err).To(MatchError(fmt.Sprintf("invalid value \"not-a-number\" for flag -download-threads: parse error")))
					})
				})

				Context("when local releases cannot be accessed", func() {
					BeforeEach(func() {
						fakeLocalReleaseDirectory.GetLocalReleasesReturns(nil, errors.New("some-error"))
					})
					It("returns an error", func() {
						err := fetch.Execute([]string{
							"--releases-directory", someReleasesDirectory,
							"--kilnfile", someKilnfilePath,
						})
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("some-error"))
					})
				})
			})
		})

	})

	Describe("Usage", func() {
		It("returns usage information for the command", func() {
			Expect(fetch.Usage()).To(Equal(jhanda.Usage{
				Description:      "Fetches releases listed in Kilnfile.lock from S3 and downloads it locally",
				ShortDescription: "fetches releases",
				Flags:            fetch.Options,
			}))
		})
	})
})
