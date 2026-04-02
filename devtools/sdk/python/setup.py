from setuptools import setup, find_packages

setup(
    name="cloudmock",
    version="0.1.0",
    description="CloudMock devtools SDK for Python",
    long_description="Connects Python applications to the CloudMock devtools for HTTP traffic, logging, and error capture.",
    author="Neureaux",
    license="MIT",
    packages=find_packages(),
    python_requires=">=3.8",
    extras_require={
        "requests": ["requests>=2.20.0"],
    },
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
    ],
)
