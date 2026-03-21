from setuptools import setup, find_packages
setup(
    name="c7n-cloudmock",
    version="0.1.0",
    packages=find_packages(),
    install_requires=["c7n"],
    entry_points={
        "custodian.plugins": ["cloudmock = c7n_cloudmock.plugin:register"]
    },
)
