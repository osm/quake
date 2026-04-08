package qwz_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/osm/quake/demo/qwz"
	"github.com/osm/quake/demo/qwz/assets"
	"github.com/osm/quake/demo/qwz/freq"
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

func TestDecodeFixtures(t *testing.T) {
	ft, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("new freq tables: %v", err)
	}

	da := assets.Assets{
		PrecacheModels:     assets.PrecacheModels,
		PrecacheSounds:     assets.PrecacheSounds,
		CenterPrintStrings: assets.EmbeddedStringTable(assets.CenterPrintStrings),
		PrintMode3Strings:  assets.EmbeddedStringTable(assets.PrintMode3Strings),
		PrintStrings:       assets.EmbeddedStringTable(assets.PrintStrings),
		SetInfoStrings:     assets.EmbeddedStringTable(assets.SetInfoStrings),
		StuffTextStrings:   assets.EmbeddedStringTable(assets.StuffTextStrings),
	}

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

			got, err := qwz.Decode(qwzData, ft, da)
			if err != nil {
				t.Fatalf("decode %s: %v", qwzPath, err)
			}

			gotChecksum := checksum(got)
			if gotChecksum != wantChecksum {
				t.Fatalf("decoded checksum mismatch: got %s want %s", gotChecksum, wantChecksum)
			}
		})
	}

	if len(fixtureChecksums) != len(qwzPaths) {
		t.Fatalf("checksum fixture count mismatch: got %d want %d", len(fixtureChecksums), len(qwzPaths))
	}
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
