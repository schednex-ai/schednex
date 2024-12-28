package prompt

var (
	// Prompt is the default prompt for the user
	Standard = "\"Given the following nodes and cluster analysis, return ONLY the name of the best node for placement. Respond with the node name directly, without any additional text or explanations.\" +\n\t\t\"Here is the data: \\n\" +\n\t\t\"1. Nodes in the cluster: %s\\n\" +\n\t\t\"2. Analysis of issues in the cluster (this may be empty): %s\\n\" +\n\t\t\"3. Relatives and their current placement: %s\\n\""
)
