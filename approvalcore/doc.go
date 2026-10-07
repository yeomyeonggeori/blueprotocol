// Package approvalcore is the sequence every gate runs on a call that needs
// approval: spend an approved hold for this exact call, otherwise open a hold,
// word the question, ask, and spend the hold once it is approved. A host or a
// harness supplies how the question is asked and the facts only it knows.
package approvalcore
