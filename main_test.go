package flokkr

import "testing"
import "github.com/stretchr/testify/assert"

func TestYamlParse(t *testing.T) {
	desc, err := ReadDescriptor()
	assert.Nil(t, err)
	assert.Equal(t, "hadoop", desc.Name)
	assert.Equal(t, "hadoop/common/hadoop-%s/hadoop-%s.tar.gz", desc.UrlPath)

	versions := desc.VersionsAndTags()
	assert.Equal(t, 3, len(versions))
	assert.Equal(t, []string{"latest", "3.2.1", "3.2", "3"}, versions["3.2.1"])
	assert.Equal(t, []string{"3.2.0"}, versions["3.2.0"])
	assert.Equal(t, []string{"3.1.2", "3.1"}, versions["3.1.2"])

}
