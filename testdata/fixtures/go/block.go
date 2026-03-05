package tsdb

// BlockStats contains statistics about a block of data.
type BlockStats struct {
	NumSamples    uint64
	NumSeries     uint64
	NumChunks     uint64
	NumTombstones uint64
	NumBytes      int64
}

// BlockDesc describes a block by its ID and time range.
type BlockDesc struct {
	ULID    string
	MinTime int64
	MaxTime int64
}

// BlockMeta is the meta information for a block of data.
type BlockMeta struct {
	ULID       string              `json:"ulid"`
	MinTime    int64               `json:"minTime"`
	MaxTime    int64               `json:"maxTime"`
	Stats      BlockStats          `json:"stats"`
	Version    int                 `json:"version"`
	Compaction BlockMetaCompaction `json:"compaction"`
}

// BlockMetaCompaction holds information about compactions a block went through.
type BlockMetaCompaction struct {
	Level   int         `json:"level"`
	Sources []string    `json:"sources,omitempty"`
	Parents []BlockDesc `json:"parents,omitempty"`
}
