#!/usr/bin/env bats

load '../node_modules/bats-support/load'
load '../node_modules/bats-assert/load'
source ../content/bundles/${CONTENT}/test/.publisher-env
CONTENT_PATH='../content/bundles'

# helper funciton for deploys
deploy_assertion() {
    if [[ ${quarto_r_content[@]} =~ ${CONTENT} ]]; then
        assert_output --partial "error detecting content type: quarto with knitr engine is not yet supported."
    else
        assert_success
        assert_output --partial "Test Deployment...                 [OK]"
        # test the deployment via api
        GUID="$(echo "${output}" | \
            grep "Direct URL:" | \
            grep -o -E '[0-9a-f-]{36}')"
        run curl --silent --show-error -L --max-redirs 0 --fail \
            -X GET \
            -H "Authorization: Key ${CONNECT_API_KEY}" "${CONNECT_SERVER}/__api__/v1/content/${GUID}"
        assert_output --partial "\"app_mode\":\"${APP_MODE}\""
    fi
}

# temporary unsupported quarto types
quarto_r_content=(
    "quarto-proj-r-shiny" "quarto-proj-r" "quarto-proj-r-py" 
    "quarty-website-r" "quarto-website-r-py" 
    "quarto-website-r-py-separate-files-deps" "quarto-website-r-deps"
    "quarto-website-r-py-deps"
    )

python_content_types=(
    "python-flask"  "python-fastapi"  "python-shiny"
     "python-bokeh"  "python-streamlit"  "python-flask"
     "jupyter-voila" "jupyter-static"
)

quarto_content_types=(
    "quarto" "quarto-static"
)
# deploy content with the env account
@test "deploy ${CONTENT}" {

    run ${EXE} deploy ${CONTENT_PATH}/${CONTENT} -n ci_deploy
    deploy_assertion
}

# redeploy content from previous test
@test "redeploy ${CONTENT}" {

    run ${EXE} redeploy ci_deploy ${CONTENT_PATH}/${CONTENT}
    deploy_assertion

    # cleanup
    run rm -rf ${CONTENT_PATH}/${CONTENT}/.posit/ ${CONTENT_PATH}/${CONTENT}/.positignore
}
