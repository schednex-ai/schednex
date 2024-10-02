package prompt

var (
	// Prompt is the default prompt for the user
	Standard = "\"Given the following nodes and analysis of issues in the cluster, I want you to tell me the best node for placement, no other text.\" +\n\t\t\"Please find the data in two segments below: \\n\" +\n\t\t\"1. Nodes in the cluster: %s\\n\" +\n\t\t\"2. Analysis of issues in the cluster (this may be empty) %s\\n\" +\n\t\t\"3. Relatives and their current placement: %s\\n\""
)
