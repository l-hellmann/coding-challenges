package null

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNullFilled(t *testing.T) {
	assert := require.New(t)

	{
		n := New("foo")
		assert.True(n.filled)
		assert.Equal("foo", n.value)
	}

	{
		n := Empty[int]()
		assert.False(n.filled)
		assert.Equal(0, n.value)
	}
}

func TestNullJsonMarshal(t *testing.T) {
	assert := require.New(t)

	type person struct {
		Name    string `json:"name"`
		Surname string `json:"surname"`
	}

	{
		n := New(person{
			Name:    "john",
			Surname: "doe",
		})
		marshal, err := json.Marshal(n)
		assert.NoError(err)
		assert.Equal(`{"name":"john","surname":"doe"}`, string(marshal))
	}

	{
		n := Empty[person]()
		marshal, err := json.Marshal(n)
		assert.NoError(err)
		assert.Equal(`null`, string(marshal))
	}
}

func TestNullJsonUnmarshal(t *testing.T) {
	assert := require.New(t)

	{
		var s Null[string]
		err := json.Unmarshal([]byte(`"lorem ipsum"`), &s)
		assert.NoError(err)
		assert.True(s.filled)
		assert.Equal("lorem ipsum", s.value)
	}

	{
		var s Null[int]
		err := json.Unmarshal([]byte(`null`), &s)
		assert.NoError(err)
		assert.False(s.filled)
		assert.Equal(0, s.value)
	}
}
