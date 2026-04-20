I am reviewing https://github.com/jlegrone/100-temporal-mistakes/pull/8 and expect to need to make lots of edits, so I have created a new branch from it (jlegrone/content) which is currently checked out.

I would like to pace myself with the review, and only focus on one broad category of mistakes at a time, one mistake at a time.

Please come up with a plan for the review, starting with sorting every mistake into a category, ordering the categories, ordering the mistakes within each category, and then walking me through each mistake one at a time.

When reviewing each mistake, I would like to inspect the rendered markdown myself. You should also look for suggestions according to:
0. TLDR section -- should be 1-2 sentences, succinct, and read clearly.
1. Style (sentence structure and format should follow closely from https://github.com/teivah/100-go-mistakes as an example)
2. Correctness. In particular, look out for potential hallucinations that cannot be backed up by primary sources, either official temporal docs https://docs.temporal.io, sdk examples https://github.com/temporalio/samples-go https://github.com/temporalio/samples-typescript, or package documentation https://pkg.go.dev/go.temporal.io/sdk). For assertions that can be fact-checked, suggest links directly to the authoritative source (using wikipedia style references).
3. Demonstration of value/impact. It should be clear why each mistake is worth avoiding.
4. How to avoid the mistake. Every mistake should have some actionable advice on how to avoid it in the first place, work around it, or address it once it has been encountered. If there is a systemic or automated approach that is possible, eg. a linter rule that prevents incorrect usage of a worker option, then I'd like to include a TODO comment to create a tool.
5. Working code examples! Where it is valuable, mistakes should include code examples. Every piece of code included in this repo must be valid/compile. Linking to external examples in official Temporal repos works, but some mistakes may need to be demonstrated with before & after code directly in this repo. Put all examples under ./examples/go/<mistake>/<filename>.go, and later we can devise a way to inline the examples in the docs so that they can be still be tested (eg. using https://github.com/temporal-community/snipsync).
6. Miscellaneous -- look out for stuff I didn't think about up front too!

-----

Avoid the tendency to think that Temporal is only for heavy weight, long-running operations. Even if a task takes a millisecond, it can be valuable to orchestrate it from a Temporal workflow; eg. to guarantee it is retried in the event of a system disruption, or that state is reconciled when the operation fails.