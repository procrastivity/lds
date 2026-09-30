# Usage

## Prerequisites

POSIX\-compatible sh and wc are available\.

## Count words

Count whitespace\-separated words in a text file\.

### Invocation and input

```
sh word-count.sh count sample.txt
```

Input: sample\.txt contains exactly four whitespace\-separated words\: alpha beta gamma delta\.

### Output

Executed from target\/\; observed stdout bytes were \`Word count\: 4\` followed by LF \(0x0a\)\, and exit status was 0\.

Operation evidence: word\-count\.sh\#L1\-L10\; sample\.txt\; verified by the fixture test invoking the documented command\.

### Limitations

Counts whitespace\-separated tokens only\; it does not parse or validate token meaning\.
