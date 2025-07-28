package config

// Config holds the relative filesystem paths and tool names used by the pgs-compare backend.
// These are currently hard-coded and can later be externalized into environment variables.
const (

    // HTTP server address
    ServerHost = "localhost"
    ServerPort = "8080"

	// Upload handler directories
	UploadRawDir       = "uploads/user_kits/raw"
	UploadProcessedDir = "uploads/user_kits/processed"

	// Download handler directory
	PGSDownloadDir     = "backend/data/pgs_files"

	// Directories for population reference genomes
	ReferenceAncestryDir = "backend/data/reference_genomes/1000G_ancestry"
	Reference23andmeDir  = "backend/data/reference_genomes/1000G_23andme"

	// Frequency files for refrence genomes
	ReferenceFreqAncestry = "backend/data/reference_genomes/1000G_ancestry/1000G_ancestry_qc.afreq"
	ReferenceFreq23andme  = "backend/data/reference_genomes/1000G_23andme/1000G_23andme_qc.afreq"
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
)
