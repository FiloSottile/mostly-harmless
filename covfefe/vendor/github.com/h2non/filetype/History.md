
v1.0.6 / 2019-01-22
===================

  * Merge pull request #55 from ivanlemeshev/master
  * Added ftypmp4v to MP4 matcher
  * Merge pull request #54 from aofei/master
  * chore: add support for Go modules
  * feat: add support for AAC (audio/aac)
  * Merge pull request #53 from lynxbyorion/check-for-docoments
  * Added checks for documents.
  * Merge pull request #51 from eriken/master
  * fixed bad mime and import paths
  * Merge pull request #50 from eriken/jpeg2000_support
  * fix import paths
  * jpeg2000 support
  * Merge pull request #47 from Ma124/master
  * Merge pull request #49 from amoore614/master
  * more robust check for .mov files
  * bugfix: reverse order of matcher key list so user registered matchers appear first
  * bugfix: store ptr to MatcherKeys in case user registered matchers are used.
  * update comment
  * Bump buffer size to 8K to allow for more custom file matching
  * refactor(readme): update package import path
  * Merge pull request #48 from kumakichi/support_msooxml
  * do not use v1
  * ok, master already changed travis
  * add fixtures, but MatchReader may not work for some msooxml files, 4096 bytes maybe not enough
  * support ms ooxml, #40
  * Fixed misspells
  * fix(travis): use string notation for matrix items
  * Merge pull request #42 from bruth/patch-2
  * refactor(travis): remove Go 1.6, add Go 1.10
  * Change maximum bytes required for detection
  * Merge pull request #36 from yiiTT/patch-1
  * Add MP4 dash and additional ISO formats
  * Merge pull request #34 from RangelReale/fix-mp4-case
  * Merge pull request #32 from yiiTT/fix-m4v
  * Fixed mp4 detection case-sensitivity according to http://www.ftyps.com/
  * Fix M4v matcher

v1.0.5 / 2017-12-12
===================

  * Merge pull request #30 from RangelReale/fix_mp4
  * Fix duplicated item in mp4 fix
  * Fix MP4 matcher, with information from http://www.file-recovery.com/mp4-signature-format.htm
  * Merge pull request #28 from ikovic/master
  * Updated file header example.

v1.0.4 / 2017-11-29
===================

  * fix: tests and document types matchers
  * refactor(docs): remove codesponsor
  * Merge pull request #26 from bienkma/master
  * Add support check file type: .doc, .docx, .pptx, .ppt, .xls, .xlsx
  * feat(docs): add code sponsor banner
  * feat(travis): add go 1.9
  * Merge pull request #24 from strazzere/patch-1
  * Fix typo in unknown

v1.0.3 / 2017-08-03
===================

  * Merge pull request #21 from elemeta/master
  * Add Elf file as supported matcher archive type

v1.0.2 / 2017-07-26
===================

  * Merge pull request #20 from marshyski/master
  * Added RedHat RPM as supported matcher archive type
  * Merge pull request #19 from nlamirault/patch-1
  * Fix typo in documentation

v1.0.1 / 2017-02-24
===================

  * Merge pull request #18 from Impyy/enable-webm
  * Enable the webm matcher
  * feat(docs): add Go version badge

1.0.0 / 2016-12-11
==================

- Initial stable version (v1.0.0).
