package main

// disable dot-imports only for testing
//revive:disable:dot-imports
import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	. "github.com/onsi/ginkgo/v2" //nolint:golint
	. "github.com/onsi/gomega"    //nolint:golint
)

// chrootTestHelper performs chroot syscall, returns func to get back to original root or error if occurred
func chrootTestHelper(path string) (func() error, error) {
	root, err := os.Open("/")
	if err != nil {
		return nil, err
	}

	if err := syscall.Chroot(path); err != nil {
		root.Close()
		return nil, err
	}

	return func() error {
		defer root.Close()
		if err := root.Chdir(); err != nil {
			return err
		}
		return syscall.Chroot(".")
	}, nil
}

func TestThinEntrypoint(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "thin_entrypoint")
}

var _ = Describe("thin entrypoint testing", func() {
	It("always pass just example", func() {
		a := 10
		Expect(a).To(Equal(10))
	})

	It("Run verifyFileExists() with expected environment, autoconfig", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		cniConfDir := fmt.Sprintf("%s/cni_conf_dir", tmpDir)
		cniBinDir := fmt.Sprintf("%s/cni_bin_dir", tmpDir)
		multusBinFile := fmt.Sprintf("%s/multus_bin", tmpDir)
		multusConfFile := fmt.Sprintf("%s/multus_conf", tmpDir)

		// CNIConfDir
		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// CNIBinDir
		Expect(os.Mkdir(cniBinDir, 0755)).To(Succeed())

		// MultusBinFile
		Expect(os.WriteFile(multusBinFile, nil, 0744)).To(Succeed())

		// MultusConfFile
		Expect(os.WriteFile(multusConfFile, nil, 0744)).To(Succeed())

		err = (&Options{
			CNIConfDir:     cniConfDir,
			CNIBinDir:      cniBinDir,
			MultusBinFile:  multusBinFile,
			MultusConfFile: multusConfFile,
		}).verifyFileExists()
		Expect(err).NotTo(HaveOccurred())

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run verifyFileExists() with invalid environmentMultusConfFile", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		cniConfDir := fmt.Sprintf("%s/cni_conf_dir", tmpDir)
		cniBinDir := fmt.Sprintf("%s/cni_bin_dir", tmpDir)
		multusBinFile := fmt.Sprintf("%s/multus_bin", tmpDir)
		multusConfFile := fmt.Sprintf("%s/multus_conf", tmpDir)

		// CNIConfDir
		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// CNIBinDir
		Expect(os.Mkdir(cniBinDir, 0755)).To(Succeed())

		// MultusConfFile
		Expect(os.WriteFile(multusConfFile, nil, 0744)).To(Succeed())

		err = (&Options{
			CNIConfDir:     cniConfDir,
			CNIBinDir:      cniBinDir,
			MultusBinFile:  multusBinFile,
			MultusConfFile: multusConfFile,
		}).verifyFileExists()
		Expect(err).To(HaveOccurred())

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run createMultusConfig(), default, conf", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		multusAutoConfigDir := fmt.Sprintf("%s/auto_conf", tmpDir)
		cniConfDir := fmt.Sprintf("%s/cni_conf", tmpDir)

		Expect(os.Mkdir(multusAutoConfigDir, 0755)).To(Succeed())
		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// create master CNI config
		masterCNIConfig := `
		{
			"cniVersion": "0.3.1",
			"name": "test1",
			"type": "cnitesttype"
		}`
		Expect(os.WriteFile(fmt.Sprintf("%s/10-testcni.conf", multusAutoConfigDir), []byte(masterCNIConfig), 0755)).To(Succeed())

		masterConfigPath, masterConfigHash, err := (&Options{
			MultusAutoconfigDir:      multusAutoConfigDir,
			CNIConfDir:               cniConfDir,
			MultusKubeConfigFileHost: "/etc/foobar_kubeconfig",
		}).createMultusConfig(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(masterConfigPath).NotTo(Equal(""))
		Expect(masterConfigHash).NotTo(Equal(""))

		expectedResult := `{
        "cniVersion": "0.3.1",
        "name": "multus-cni-network",
        "type": "multus",
        "logToStderr": false,
        "kubeconfig": "/etc/foobar_kubeconfig",
        "delegates": [
                {"cniVersion":"0.3.1","name":"test1","type":"cnitesttype"}
        ]
}
`
		conf, err := os.ReadFile(fmt.Sprintf("%s/00-multus.conf", cniConfDir))
		Expect(string(conf)).To(Equal(expectedResult))

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run createMultusConfig(), capabilities, conf", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		multusAutoConfigDir := fmt.Sprintf("%s/auto_conf", tmpDir)
		cniConfDir := fmt.Sprintf("%s/cni_conf", tmpDir)

		Expect(os.Mkdir(multusAutoConfigDir, 0755)).To(Succeed())

		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// create master CNI config
		masterCNIConfig := `
		{
			"cniVersion": "0.3.1",
			"name": "test1",
			"capabilities": { "bandwidth": true },
			"type": "cnitesttype"
		}`
		Expect(os.WriteFile(fmt.Sprintf("%s/10-testcni.conf", multusAutoConfigDir), []byte(masterCNIConfig), 0755)).To(Succeed())

		masterConfigPath, masterConfigHash, err := (&Options{
			MultusAutoconfigDir:      multusAutoConfigDir,
			CNIConfDir:               cniConfDir,
			MultusKubeConfigFileHost: "/etc/foobar_kubeconfig",
		}).createMultusConfig(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(masterConfigPath).NotTo(Equal(""))
		Expect(masterConfigHash).NotTo(Equal(""))

		expectedResult := `{
        "cniVersion": "0.3.1",
        "name": "multus-cni-network",
        "type": "multus",
        "capabilities": {"bandwidth":true},
        "logToStderr": false,
        "kubeconfig": "/etc/foobar_kubeconfig",
        "delegates": [
                {"capabilities":{"bandwidth":true},"cniVersion":"0.3.1","name":"test1","type":"cnitesttype"}
        ]
}
`
		conf, err := os.ReadFile(fmt.Sprintf("%s/00-multus.conf", cniConfDir))
		Expect(string(conf)).To(Equal(expectedResult))

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run createMultusConfig(), with options, conf", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		multusAutoConfigDir := fmt.Sprintf("%s/auto_conf", tmpDir)
		cniConfDir := fmt.Sprintf("%s/cni_conf", tmpDir)

		Expect(os.Mkdir(multusAutoConfigDir, 0755)).To(Succeed())

		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// create master CNI config
		masterCNIConfig := `
		{
			"cniVersion": "0.3.1",
			"name": "test1",
			"type": "cnitesttype"
		}`
		err = os.WriteFile(fmt.Sprintf("%s/10-testcni.conf", multusAutoConfigDir), []byte(masterCNIConfig), 0755)
		Expect(err).NotTo(HaveOccurred())

		masterConfigPath, masterConfigHash, err := (&Options{
			MultusAutoconfigDir:      multusAutoConfigDir,
			CNIConfDir:               cniConfDir,
			MultusKubeConfigFileHost: "/etc/foobar_kubeconfig",
			NamespaceIsolation:       true,
			GlobalNamespaces:         "foobar,barfoo",
			MultusLogToStderr:        true,
			MultusLogLevel:           "DEBUG",
			MultusLogFile:            "/tmp/foobar.log",
			AdditionalBinDir:         "/tmp/add_bin_dir",
			MultusCNIConfDir:         "/tmp/multus/net.d",
			ReadinessIndicatorFile:   "/var/lib/foobar_indicator",
		}).createMultusConfig(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(masterConfigPath).NotTo(Equal(""))
		Expect(masterConfigHash).NotTo(Equal(""))

		expectedResult := `{
        "cniVersion": "0.3.1",
        "name": "multus-cni-network",
        "type": "multus",
        "namespaceIsolation": true,
        "globalNamespaces": "foobar,barfoo",
        "logLevel": "debug",
        "logFile": "/tmp/foobar.log",
        "binDir": "/tmp/add_bin_dir",
        "cniConf": "/tmp/multus/net.d",
        "readinessindicatorfile": "/var/lib/foobar_indicator",
        "kubeconfig": "/etc/foobar_kubeconfig",
        "delegates": [
                {"cniVersion":"0.3.1","name":"test1","type":"cnitesttype"}
        ]
}
`
		conf, err := os.ReadFile(fmt.Sprintf("%s/00-multus.conf", cniConfDir))
		Expect(string(conf)).To(Equal(expectedResult))

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run createMultusConfig(), default, conflist", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		multusAutoConfigDir := fmt.Sprintf("%s/auto_conf", tmpDir)
		cniConfDir := fmt.Sprintf("%s/cni_conf", tmpDir)

		Expect(os.Mkdir(multusAutoConfigDir, 0755)).To(Succeed())
		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// create master CNI config
		masterCNIConfig := `
		{
			"cniVersion": "1.0.0",
			"name": "test1",
			"type": "cnitesttype"
		}`
		Expect(os.WriteFile(fmt.Sprintf("%s/10-testcni.conf", multusAutoConfigDir), []byte(masterCNIConfig), 0755)).To(Succeed())

		masterConfigPath, masterConfigHash, err := (&Options{
			MultusAutoconfigDir:      multusAutoConfigDir,
			CNIConfDir:               cniConfDir,
			MultusKubeConfigFileHost: "/etc/foobar_kubeconfig",
		}).createMultusConfig(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(masterConfigPath).NotTo(Equal(""))
		Expect(masterConfigHash).NotTo(Equal(""))

		expectedResult :=
			`{
    "cniVersion": "1.0.0",
    "name": "multus-cni-network",
    "plugins": [ {
        "type": "multus",
        "logToStderr": false,
        "kubeconfig": "/etc/foobar_kubeconfig",
        "delegates": [
            {"cniVersion":"1.0.0","name":"test1","type":"cnitesttype"}
        ]
    }]
}
`
		conf, err := os.ReadFile(fmt.Sprintf("%s/00-multus.conflist", cniConfDir))
		Expect(string(conf)).To(Equal(expectedResult))

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run createMultusConfig(), capabilities, conflist", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		multusAutoConfigDir := fmt.Sprintf("%s/auto_conf", tmpDir)
		cniConfDir := fmt.Sprintf("%s/cni_conf", tmpDir)

		Expect(os.Mkdir(multusAutoConfigDir, 0755)).To(Succeed())
		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// create master CNI config
		masterCNIConfig := `
		{
			"cniVersion": "1.0.0",
			"name": "test1",
			"capabilities": { "bandwidth": true },
			"type": "cnitesttype"
		}`
		Expect(os.WriteFile(fmt.Sprintf("%s/10-testcni.conflist", multusAutoConfigDir), []byte(masterCNIConfig), 0755)).To(Succeed())

		masterConfigPath, masterConfigHash, err := (&Options{
			MultusAutoconfigDir:      multusAutoConfigDir,
			CNIConfDir:               cniConfDir,
			MultusKubeConfigFileHost: "/etc/foobar_kubeconfig",
		}).createMultusConfig(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(masterConfigPath).NotTo(Equal(""))
		Expect(masterConfigHash).NotTo(Equal(""))

		expectedResult :=
			`{
    "cniVersion": "1.0.0",
    "name": "multus-cni-network",
    "plugins": [ {
        "type": "multus",
        "capabilities": {"bandwidth":true},
        "logToStderr": false,
        "kubeconfig": "/etc/foobar_kubeconfig",
        "delegates": [
            {"capabilities":{"bandwidth":true},"cniVersion":"1.0.0","name":"test1","type":"cnitesttype"}
        ]
    }]
}
`
		conf, err := os.ReadFile(fmt.Sprintf("%s/00-multus.conflist", cniConfDir))
		Expect(string(conf)).To(Equal(expectedResult))

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run createMultusConfig(), with options, conflist", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		multusAutoConfigDir := fmt.Sprintf("%s/auto_conf", tmpDir)
		cniConfDir := fmt.Sprintf("%s/cni_conf", tmpDir)

		Expect(os.Mkdir(multusAutoConfigDir, 0755)).To(Succeed())
		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// create master CNI config
		masterCNIConfig := `
		{
			"cniVersion": "1.0.0",
			"name": "test1",
			"type": "cnitesttype"
		}`
		Expect(os.WriteFile(fmt.Sprintf("%s/10-testcni.conflist", multusAutoConfigDir), []byte(masterCNIConfig), 0755)).To(Succeed())

		masterConfigPath, masterConfigHash, err := (&Options{
			MultusAutoconfigDir:      multusAutoConfigDir,
			CNIConfDir:               cniConfDir,
			MultusKubeConfigFileHost: "/etc/foobar_kubeconfig",
			NamespaceIsolation:       true,
			GlobalNamespaces:         "foobar,barfoo",
			MultusLogToStderr:        true,
			MultusLogLevel:           "DEBUG",
			MultusLogFile:            "/tmp/foobar.log",
			AdditionalBinDir:         "/tmp/add_bin_dir",
			MultusCNIConfDir:         "/tmp/multus/net.d",
			ReadinessIndicatorFile:   "/var/lib/foobar_indicator",
		}).createMultusConfig(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(masterConfigPath).NotTo(Equal(""))
		Expect(masterConfigHash).NotTo(Equal(""))

		expectedResult :=
			`{
    "cniVersion": "1.0.0",
    "name": "multus-cni-network",
    "plugins": [ {
        "type": "multus",
        "namespaceIsolation": true,
        "globalNamespaces": "foobar,barfoo",
        "logLevel": "debug",
        "logFile": "/tmp/foobar.log",
        "binDir": "/tmp/add_bin_dir",
        "cniConf": "/tmp/multus/net.d",
        "readinessindicatorfile": "/var/lib/foobar_indicator",
        "kubeconfig": "/etc/foobar_kubeconfig",
        "delegates": [
            {"cniVersion":"1.0.0","name":"test1","type":"cnitesttype"}
        ]
    }]
}
`
		conf, err := os.ReadFile(fmt.Sprintf("%s/00-multus.conflist", cniConfDir))
		Expect(string(conf)).To(Equal(expectedResult))

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run createMultusConfig(), with options, conflist", func() {
		// create directory and files
		tmpDir, err := os.MkdirTemp("", "multus_thin_entrypoint_tmp")
		Expect(err).NotTo(HaveOccurred())

		multusAutoConfigDir := fmt.Sprintf("%s/auto_conf", tmpDir)
		cniConfDir := fmt.Sprintf("%s/cni_conf", tmpDir)

		Expect(os.Mkdir(multusAutoConfigDir, 0755)).To(Succeed())
		Expect(os.Mkdir(cniConfDir, 0755)).To(Succeed())

		// create master CNI config
		masterCNIConfigFileName := "10-testcni.conf"
		masterCNIConfig := `
		{
			"cniVersion": "1.0.0",
			"name": "test1",
			"type": "cnitesttype"
		}`
		Expect(os.WriteFile(fmt.Sprintf("%s/%s", multusAutoConfigDir, masterCNIConfigFileName), []byte(masterCNIConfig), 0755)).To(Succeed())

		// create another CNI config
		anotherCNIConfigFileName := "09-test2cni.conf" // Alphabetically before masterCNIConfigFileName
		anotherCNIConfig := `
		{
			"cniVersion": "1.0.0",
			"name": "test2",
			"type": "cnitest2type"
		}`
		Expect(os.WriteFile(fmt.Sprintf("%s/%s", multusAutoConfigDir, anotherCNIConfigFileName), []byte(anotherCNIConfig), 0755)).To(Succeed())

		masterConfigPath, masterConfigHash, err := (&Options{
			MultusAutoconfigDir:      multusAutoConfigDir,
			MultusMasterCNIFileName:  masterCNIConfigFileName,
			CNIConfDir:               cniConfDir,
			MultusKubeConfigFileHost: "/etc/foobar_kubeconfig",
		}).createMultusConfig(nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(masterConfigPath).NotTo(Equal(""))
		Expect(masterConfigHash).NotTo(Equal(""))

		expectedResult :=
			`{
    "cniVersion": "1.0.0",
    "name": "multus-cni-network",
    "plugins": [ {
        "type": "multus",
        "logToStderr": false,
        "kubeconfig": "/etc/foobar_kubeconfig",
        "delegates": [
            {"cniVersion":"1.0.0","name":"test1","type":"cnitesttype"}
        ]
    }]
}
`
		conf, err := os.ReadFile(fmt.Sprintf("%s/00-multus.conflist", cniConfDir))
		Expect(string(conf)).To(Equal(expectedResult))

		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("Run createKubeConfig()", func() {
		// create temp dir and files
		tmpDir := GinkgoT().TempDir()

		cniConfDir := "/cni_conf"
		Expect(os.Mkdir(filepath.Join(tmpDir, cniConfDir), 0755)).To(Succeed())

		multusConfDir := "/multus_conf"
		Expect(os.Mkdir(filepath.Join(tmpDir, multusConfDir), 0755)).To(Succeed())

		// Create service account CA file and token file with dummy data
		svcAccountPath := filepath.Join(tmpDir, "var/run/secrets/kubernetes.io/serviceaccount")
		Expect(os.MkdirAll(svcAccountPath, 0755)).ToNot(HaveOccurred())
		svcAccountCAFile := filepath.Join(tmpDir, serviceAccountCAFile)
		svcAccountTokenFile := filepath.Join(tmpDir, serviceAccountTokenFile)
		Expect(os.WriteFile(svcAccountCAFile, []byte("dummy-ca-content"), 0644)).To(Succeed())
		Expect(os.WriteFile(svcAccountTokenFile, []byte("dummy-token-content"), 0644)).To(Succeed())

		// Set up the Options struct
		options := &Options{
			CNIConfDir:       cniConfDir,
			MultusCNIConfDir: multusConfDir,
		}

		// Run the createKubeConfig function in a chroot env
		back, err := chrootTestHelper(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		caHash, saTokenHash, err := options.createKubeConfig(nil, nil)
		Expect(back()).ToNot(HaveOccurred())
		// back to original root

		Expect(err).NotTo(HaveOccurred())
		Expect(caHash).NotTo(BeNil())
		Expect(saTokenHash).NotTo(BeNil())

		// Verify the kubeconfig file was created successfully
		kubeConfigPath := filepath.Join(tmpDir, cniConfDir, "multus.d", "multus.kubeconfig")
		content, err := os.ReadFile(kubeConfigPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(content).NotTo(BeEmpty())

		// Cleanup
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

})
