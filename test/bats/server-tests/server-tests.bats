#!/usr/bin/env bats

${BATS_SUPPORT_LIB}
${BATS_ASSERT_LIB}

@test "${BINARY_PATH} list servers" {
    run ${BINARY_PATH} list-accounts
    assert_success
    assert_output --partial "No accounts are saved. To add an account, see \`connect-client add-server --help\`."
}