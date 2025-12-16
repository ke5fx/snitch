# Golden Files

This directory contains golden files for output contract verification tests.

These files are automatically generated and should not be edited manually.
To regenerate them, run:

    go test ./cmd -update-golden

## Files

- *_table.golden: Table format output
- *_json.golden: JSON format output  
- *_csv.golden: CSV format output
- *_wide.golden: Wide table format output
- stats_*.golden: Statistics command output

Each file represents expected output for specific test scenarios.
