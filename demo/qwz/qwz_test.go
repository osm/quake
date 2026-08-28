package qwz_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/osm/quake/demo/qwz"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
)

var fixtureChecksums = map[string]string{
	"demo00": "9fecb0607e113ad5a88caca843fbc601191eb9efa3a92d030894b9e78c08b02b",
	"demo01": "042ade84756331421deaf7067fafc61208ab2b69b4893c12fd41a85c6fc10ddb",
	"demo02": "07bbea552eb74d936a222adc3c4b1abca382ac54ebabc86a94cb022793620d44",
	"demo03": "514db355d15a5bb860f5575115e387d45c590314213bcaf013334b64ef9b3bf9",
	"demo04": "0a1a391a6f38f16321d0ac36008f0f047b05b230f389f8c31aedafdc17f8871e",
	"demo05": "d9843a0946f51575535bdaae09a9434cf4edcc8bab73dea5b1775a557c29d938",
	"demo06": "f7114293faba20246c47fe6548401208908e1fd7090098f1271d8066ab9386c6",
	"demo07": "827078b59dcef16efd74b86b7a550ca3a3d05b69159ca3dd71866ba57c58334b",
	"demo08": "3488a0d20ba5a4b90d57343f8b4e3ac6c2c7a5eef323b6721240b480895e5a42",
	"demo09": "4773551355503c36ba92a62e7e53a1cdb09ebdc328ec5bac2d2523b1ddef7394",
	"demo10": "31f13ca65fbdbb73d943574548bf5218cbe5335fb4a0e07a594b439b788fbe03",
	"demo11": "12d0631cdf58fc72b9813b6ef8b5f34efa457d0dafbe262ca51bc9718f0771cb",
	"demo12": "454f351c92240ded39a38e3d0b83310c05c866c9ab66f71055a6d9042bfc928a",
	"demo13": "51df63f9603bba18bf0862604ea4bbc5b0de2dbdce5c3bf8789e9f95386b601e",
	"demo14": "b383ac3417f3a92c19a6ea4564afe1c917e7ed7a215b6daeee0e26be0dd40ced",
	"demo15": "aaa8d2d0036ea37a333e9a2cb7b16f9a1f2edc345e6521a8ccc99a5953632dce",
	"demo16": "ca9dcf0d6eb18f6a77f7676b5175d4c7b8ec4191500fa782a1f76a21f844c79a",
	"demo17": "28e05fad75cacf4577f7c5ae0e1fa19fa4f36fa960005ca908bb011bc20a73de",
	"demo18": "a9b483a1f9e679de08078859ea6004bf6ff3060b1fac235eb6baa0141d263987",
	"demo19": "d34164ceca30be2b6fa661463cdac95564801951db9d3c5ce997b5142fbba90e",
	"demo20": "fc4f3e3b0b929951144bb8fd5702d82093b2a3d6229b86719918b382f117d2b1",
	"demo21": "6cf18d4ca19b1cff2c96b25d6f49edd7c0701a02719637990fe844f49b5686c8",
	"demo22": "c08a39dbe1f4e6bf6c96d4377a9d440f1f98e2b5db3e44e5eff07550ccee3663",
	"demo23": "a1fa879405e5779cdb79dcb8965ed8f743d87da3af7effbce832f2512cca7dbe",
	"demo24": "ac7fac5b9717e2a8f9d60f6f4c4f497145545425a7f4541adb9b8bdc5dfd611f",
	"demo25": "c400e5a37cd6494d98de8958c07b5f1edf2705f6acc98066074cb3831aaf31b9",
	"demo26": "888dc6b7742829c453500575a69f960bb3af17fd417e4e2c447c2589ae44d33a",
	"demo27": "f9b701cfffe7acfa3e5458bf29d2124785b587cc127c4ce971976884743a9919",
	"demo28": "1a2a02acbfb0c8df726372b53534854ce3f5f991b695fab01fd38884252aeb32",
	"demo29": "b83980edcce270c22f96d3f6ed821003b9ccf7d26884c3fe41494a164dc31460",
	"demo30": "1b2d4859a274bb3921c4ac9a2ac6457f48a569a6359a919c8c9ef4115408b8f0",
	"demo31": "5e065f88b8d316a6f12010484122d1bcb7a8768d5fceb76be0614c1d9154a244",
	"demo32": "d41534c265f27a17aa0ce491b149c421b426efa713006497d368c83cef3ae3be",
	"demo33": "aaa8d2d0036ea37a333e9a2cb7b16f9a1f2edc345e6521a8ccc99a5953632dce",
	"demo34": "6e80a881742eb0123ec9257e964e56c1f2c6fb402dde37d9d51e3de4cbe19c24",
	"demo35": "85d4acb70496b08c6a0589d7a4d7c8ec9e192d1ea355b52458c0cc086fe96bb5",
	"demo36": "81523f6a217afd285eabf08a18aa79575b175fc8a51e3d7c72336c9d6661dfb5",
	"demo37": "5f14d04f09d6196bdd87dec0c0119d7b159ea659d0ff139073bedd4408f74edd",
	"demo38": "3d723aa0749a13fed49aa361750b74d9c7eb3759bd996d1f7cac642aa58afc96",
}

// encodeChecksums were produced by running each fixture through a fresh
// original Qizmo process (-D, followed by -C in another fresh process).
// demo36 is absent because the original compressor terminates without output.
var encodeChecksums = map[string]string{
	"demo00": "4a922f2dcc8d127e77583a8138495a02a71cbf44a5a65196af58d32bbacb1a91",
	"demo01": "e6e3a4cc57e42a9ec4d0a5d05cdb381dd469b209498b45575c03aec572425a70",
	"demo02": "5364fbe43ec2a5295a1673de89b641277255d61302c92d366d34235938366fca",
	"demo03": "1e3b9561d50ee9b1a15dba4a3a5c59e006393317626838e9d4d8c224c9efa1db",
	"demo04": "0b48d9b9d8a99a0c5a0f5c0f75f325d26f10b64f08f6cec121696e8f81ce0dbf",
	"demo05": "559a1aed03f8c65b97af6594a4f3aafeb9d3b6a409ff10e9214057c403a2ab4e",
	"demo06": "d93c51ca9e24c6d67911329dcfe20901f6ee0111ec02f4d86b5d7a4041ebc42e",
	"demo07": "59ac00b2b27c5c47e9ea8f31814a528b947a6ecd16e3253736bcf7cd37d92a7d",
	"demo08": "18aa7acf8f04aecd17b8dee2c081bbbd54b10b9f4cde48e1309d472538e59d20",
	"demo09": "423dc2297dd8a49d20312771dc0520087ba947fe92c8f4353ee69a6c8e814a5a",
	"demo10": "57e13bcc3954048db6ff4896871da24d6b9bc8b34c44d87b6734a14711592beb",
	"demo11": "88dc47d6b8396f5721490d2a1fc116d9f5ad3a22d0818b3db89126818e5f79a9",
	"demo12": "fdda3451d106b47ad175fcecc2fa902c6d3f76130e812def761ca858278a8265",
	"demo13": "3f80a911557d5955811986aaa843228c28e3e221c3a3b43658acb4f57d9fad9b",
	"demo14": "2ac0e7fd19ebbb277988393baccd65714180ea80886afde3ffda9d14e56235f8",
	"demo15": "7820058f456925da0a3bed6a349677c1f5092675a10b06c2cb2d54e106eb9898",
	"demo16": "e3895fce36fc8b85f31664aadba7cd7c6ab665f54509a828d15c582208097863",
	"demo17": "1efd06c460cc1e3f95d2160d146f7a9ef4abfb3a83aa028ecfbae9b170d4b670",
	"demo18": "74a0b09a510c2cb6a6fe1b68c5dbd6b2c3af95ec271d28b322d2cf29838e342f",
	"demo19": "fbe403ea883387ede8ef4f7ab1911b440fd430405f6d76348f240d005ac719bf",
	"demo20": "9a809a34613e4f55097b7fc7fa33325e27af0694fc0a2697bc111a1489131aeb",
	"demo21": "786967717938e02a0197668a5a62b3884e5b36f0e2c58d6ee13b320a92d57b10",
	"demo22": "bfe339b1e347fcbb2a76c4dd223bb71150ff20a1834a1d71c9b580c8089aab8a",
	"demo23": "890a33b03c5bc9be2e332479c0dd476373a0f59e2f66bbddb1506dc3a6399303",
	"demo24": "35385e5c888dff91553f3e665d69101ba6bc381b1d191efb0aacbfbd96f2fed0",
	"demo25": "eeb8995fe1a43ea13b623979cd5f1f8489c1b4974f22314b1ac252581100e8dd",
	"demo26": "671deeced563977e896938cee23c4bb4545ffa06a0a07b98564c95bb4798d74c",
	"demo27": "368d78bd054fa62ed45e32eb390c5545a48eacf9602eeb385a15e88e59dc8370",
	"demo28": "7cd730ecce9541ac1199b55cbe08c6ff09fa75f1bb7e77176a39f9154355b1d7",
	"demo29": "fff791964e267c957adaa45ea7f348430399281a1a05a8772a3008b1b3706538",
	"demo30": "3021173c2987c932e7f59c9834df58e106419ef6b263e19aa6d67526e6de3981",
	"demo31": "d5b298c3548c3ff5d5c471c67539979512d0c1d23e8e8e534b2bdff0f834e43a",
	"demo32": "aef8764fa09d42c39338390c578084b545bbbc1eb7e341a3a4d16c58722f965f",
	"demo33": "7820058f456925da0a3bed6a349677c1f5092675a10b06c2cb2d54e106eb9898",
	"demo34": "0f38c93da484da8261d438bd4eb3523814037f84c6251690e4910bacac9d25ef",
	"demo35": "b6dd1e7c0b274f193cb0332d20a78b8fba01e99af47c3b25c8dce4d9cb5de835",
	"demo37": "d92c150140933e5a1de9a226c3a0b3519dcf6d11fcb83a7630a6fd96e2506da6",
	"demo38": "da1719b4b761f4e50e9064753870b7e7c4806fe56c9e03695dd6195403396d95",
}

func TestDecodeFixtures(t *testing.T) {
	ft, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("new freq tables: %v", err)
	}

	packetAssets := assets.Embedded()

	qwzPaths, err := filepath.Glob(filepath.Join("testdata", "*.qwz"))
	if err != nil {
		t.Fatalf("glob testdata demos: %v", err)
	}

	sort.Strings(qwzPaths)

	if len(qwzPaths) == 0 {
		t.Fatal("no .qwz fixtures found in testdata")
	}

	for _, qwzPath := range qwzPaths {
		name := strings.TrimSuffix(filepath.Base(qwzPath), ".qwz")
		wantChecksum, ok := fixtureChecksums[name]
		if !ok {
			t.Fatalf("missing checksum fixture for %s", name)
		}

		t.Run(name, func(t *testing.T) {
			qwzData, err := os.ReadFile(qwzPath)
			if err != nil {
				t.Fatalf("read %s: %v", qwzPath, err)
			}

			got, err := qwz.Decode(qwzData, ft, packetAssets)
			if err != nil {
				t.Fatalf("decode %s: %v", qwzPath, err)
			}

			gotChecksum := checksum(got)
			if gotChecksum != wantChecksum {
				t.Fatalf("decoded checksum mismatch: got %s want %s", gotChecksum, wantChecksum)
			}

			encoded, encodeErr := qwz.Encode(got, ft)
			if name == "demo36" {
				if encodeErr == nil {
					t.Fatal("Qizmo's zero-length DEMO_READ was accepted")
				}
			} else if encodeErr != nil {
				t.Fatalf("encode decoded QWD: %v", encodeErr)
			} else if wantEncodeChecksum, ok := encodeChecksums[name]; ok {
				if gotChecksum := checksum(encoded); gotChecksum != wantEncodeChecksum {
					t.Fatalf("Qizmo checksum mismatch: got %s want %s", gotChecksum, wantEncodeChecksum)
				}
				// This fixture is stable through Qizmo's decode/encode cycle, so it
				// also compares the complete archive directly.
				if name == "demo01" && !bytes.Equal(encoded, qwzData) {
					t.Fatalf("QWZ mismatch: got %d bytes, want %d", len(encoded), len(qwzData))
				}
			} else {
				t.Fatalf("missing Qizmo checksum fixture for %s", name)
			}
			// These integration fixtures are large enough that retaining several
			// generations before the next automatic collection can exhaust small
			// CI workers.
			runtime.GC()
		})
	}

	if len(fixtureChecksums) != len(qwzPaths) {
		t.Fatalf("checksum fixture count mismatch: got %d want %d", len(fixtureChecksums), len(qwzPaths))
	}
	if len(encodeChecksums) != len(qwzPaths)-1 {
		t.Fatalf("encode checksum fixture count mismatch: got %d want %d", len(encodeChecksums), len(qwzPaths)-1)
	}
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
