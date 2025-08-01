package config

// Config holds the relative filesystem paths and tool names used by the pgs-compare backend.
// These are currently hard-coded and can later be externalized into environment variables.
const (

    // HTTP server address
    ServerHost = "localhost"
    ServerPort = "8080"
	
    // DataDir is where kits.db and other local data live
    DataDir = "backend/data"

    // FrontendOrigin is the allowed origin for CORS
    FrontendOrigin = "http://localhost:3000"

	// Upload handler directories
	UploadRawDir       = "uploads/user_kits/raw"
	UploadProcessedDir = "uploads/user_kits/processed"

	// Download handler directory
	PGSDownloadDir     = "backend/data/pgs_files"

	// Directories for population reference genomes
	ReferenceAncestryDir = "backend/data/reference_genomes/1000G/ancestry"
	Reference23andmeDir  = "backend/data/reference_genomes/1000G/23andme"

	// Frequency files for reference genomes
	ReferenceFreqAncestry = ReferenceAncestryDir + "/ancestry.afreq"
	ReferenceFreq23andme  = Reference23andmeDir  + "/23andme.afreq"
	ReferenceFreqFiltered = "backend/data/reference_genomes/1000G_filtered/1000G_chip_qc.afreq"

	// Download dir for PGS files
	PGSFilesDir          = "backend/data/pgs_files"

	// DNA Kit manifest directories
	ChipManifestAncestryDir = "backend/data/dna_chip_manifests/ancestry_v2"
	ChipManifestV5Dir       = "backend/data/dna_chip_manifests/23andme_v5"

	// Score output subdirectory within each kit folder
	ScoreOutputDirName = "scores"

	// External tool binaries
	Plink1Cmd = "plink1"
	Plink2Cmd = "plink2"

	// Catalog file names
	OntologyTraitsFile = "ontology_traits.json"
  	ScoresMetadataFile = "scores_metadata.json"

	// Bolt DB
	BoltDBName     = "kits.db"
  	BoltBucketName = "kits"
)
